package server

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"
	"zmtwc/sk/internal/auth"

	"github.com/gorilla/mux"
)

type Story struct {
    ID int64
    Title string
    StartTime string
    Description string
    Creator string
    IsStoryOwner bool
}

type StoryDetail struct {
    IsUserLoggedIn bool
    Story Story
    Tasks []Task
}

type Task struct {
    IsUserLoggedIn bool
    IsStoryOwner bool
    HasJoined bool
    ID int64
    Name string
    Description string
    SlotsTotal int64
    SlotsAssigned int64
    AssignmentList []Assignments
}

type Assignments struct {
    ID int64
    AssigneeID int64
    AssigneeName string
}

func GetTaskAssignments (db *sql.DB, taskID int64, userID int64) ([]Assignments, bool, error) {
    assignments := []Assignments{}
    rows, err := db.Query(`
        SELECT
            assignment.id,
            assignment.assignee_id,
            user.username
        FROM assignment
        JOIN user ON assignment.assignee_id = user.id
        WHERE assignment.task_id = $1
        `,
        taskID,
    )
    if err != nil {
        return []Assignments{}, false, err
    }
    defer rows.Close()

    hasJoined := false
    for rows.Next() {
        var id int64
        var assigneeID int64
        var assigneeName string

        err = rows.Scan(&id, &assigneeID, &assigneeName)
        if err != nil {
            return []Assignments{}, false, err
        }
        hasJoined = hasJoined || assigneeID == userID

        assignments = append(assignments, Assignments {
            ID: id,
            AssigneeID: assigneeID,
            AssigneeName: assigneeName,
        })
    }

    return assignments, hasJoined, nil
}

func GetSingleTask (db *sql.DB, taskID int64, userID int64) (Task, error) {
    row := db.QueryRow(`
        SELECT
            task.id,
            task.name,
            task.description,
            task.slots,
            story.creator_id
        FROM task
        JOIN story ON task.story_id = story.id
        WHERE task.id = $1
        `,
        taskID,
    )

    var id int64
    var name string
    var storyOwnerID int64
    var description string
    var slots int64
    err := row.Scan(&id, &name, &description, &slots, &storyOwnerID)
    if err != nil {
        return Task{}, err
    }
    assignments, hasJoined, err := GetTaskAssignments(db, id, userID)
    if err != nil {
        return Task{}, err
    }

    task := Task {
        ID: id,
        SlotsTotal: slots,
        SlotsAssigned: int64(len(assignments)),
        Description: description,
        Name: name,
        HasJoined: hasJoined,
        AssignmentList: assignments,
        IsStoryOwner: storyOwnerID == userID,
    }

    return task, nil
}

func GetStoryTasks (db *sql.DB, storyID int64, userID int64, isStoryOwner bool, isUserLoggedIn bool) ([]Task, error) {
    tasks := []Task{}
    rows, err := db.Query(`
        SELECT
            task.id,
            task.name,
            task.description,
            task.slots
        FROM task
        WHERE task.story_id = $1
        `,
        storyID,
    )
    if err != nil {
        return []Task{}, err
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var name string
        var description string
        var slots int64

        err = rows.Scan(&id, &name, &description, &slots)
        if err != nil {
            return []Task{}, err
        }

        assignments, hasJoined, err := GetTaskAssignments(db, id, userID)
        if err != nil {
            return []Task{}, err
        }

        tasks = append(tasks, Task {
            ID: id,
            SlotsTotal: slots,
            SlotsAssigned: int64(len(assignments)),
            Description: description,
            Name: name,
            IsStoryOwner: isStoryOwner,
            IsUserLoggedIn: isUserLoggedIn,
            HasJoined: hasJoined,
            AssignmentList: assignments,
        })
    }

    return tasks, nil
}

func GetStoryData(db *sql.DB, storyID int64, userID int64) (Story, error) {
    row := db.QueryRow(`
        SELECT
            story.id,
            story.title,
            user.username,
            story.creator_id,
            story.description,
            story.start_time
        FROM story
        JOIN user on story.creator_id = user.id
        WHERE story.status > 0
        AND story.id = $1
        `,
        storyID,
    )

    var id int64
    var title string
    var creatorName string
    var creatorID int64
    var descriptionOption sql.NullString
    var startTimeOption sql.NullInt64

    err := row.Scan(&id, &title, &creatorName, &creatorID, &descriptionOption, &startTimeOption)
    if err != nil {
        return Story{}, err
    }

    description := ""
    if descriptionOption.Valid {
        description = descriptionOption.String
    }
    startTime := ""
    if startTimeOption.Valid {
        startTime = time.Unix(startTimeOption.Int64, 0).Format("02. 01. 2006 15:04")
    }

    return Story{
        ID: id,
        Title: title,
        Description: description,
        StartTime: startTime,
        Creator: creatorName,
        IsStoryOwner: creatorID == userID,
    }, nil
}

type StoryEditPageData struct {
    ID int64
    Title string
    StartTime string
    Description string
}

func StoryEditPageHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    _, _, err = auth.ValidateSession(db, r);
    if err != nil {
        http.Error(w, "Cannot find valid session", 401)
        return
    }
    row := db.QueryRow(`
        SELECT
            story.id,
            story.title,
            story.description,
            story.start_time
        FROM story
        WHERE story.id = $1`,
        storyID,
    )

    var id int64
    var title string
    var descriptionOption sql.NullString
    var startTime int64

    err = row.Scan(&id, &title, &descriptionOption, &startTime)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error loading data from database: %s", err), 500)
        return
    }

    description := ""
    if descriptionOption.Valid {
        description = descriptionOption.String
    }
    startTimeString := time.Unix(startTime, 0).Format("2006-01-02T15:04")

    tmpl := template.Must(template.ParseFiles("app/templates/story-detail.html", "app/templates/create-story.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "story-detail-edit", StoryEditPageData {
        ID: id,
        Title: title,
        Description: description,
        StartTime: startTimeString,
    })
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func StoryDetailHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);

    story, err := GetStoryData(db, storyID, userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting story: %s", err), 500)
        return
    }
    isUserLoggedIn := sessionErr == nil
    tasks, err := GetStoryTasks(db, storyID, userID, story.IsStoryOwner, isUserLoggedIn)

    tmpl := template.Must(template.ParseFiles("app/templates/story-detail.html", "app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.Execute(w, StoryDetail {
        IsUserLoggedIn: isUserLoggedIn,
        Story: story,
        Tasks: tasks,
    })
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

type StoryListData struct {
    Stories []Story
    IsUserLoggedIn bool
}

func StoryListHandler (w http.ResponseWriter, r *http.Request) {
    stories := []Story{}

    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);

    rows, err := db.Query(`
        SELECT
            story.id,
            story.title,
            user.username,
            story.creator_id,
            story.description,
            story.start_time
        FROM story
        JOIN user on story.creator_id = user.id
        WHERE story.status > 0
    `)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting story list: %s", err), 500)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var title string
        var creatorName string
        var creatorID int64
        var descriptionOption sql.NullString
        var startTimeOption sql.NullInt64

        err = rows.Scan(&id, &title, &creatorName, &creatorID, &descriptionOption, &startTimeOption)
        if err != nil {
            http.Error(w, fmt.Sprintf("Error getting story entries: %s", err), 500)
            return
        }

        description := ""
        if descriptionOption.Valid {
            description = descriptionOption.String
        }
        startTime := ""
        if startTimeOption.Valid {
            startTime = time.Unix(startTimeOption.Int64, 0).Format("02. 01. 2006 15:04")
        }

        isStoryOwner := sessionErr == nil && userID == creatorID
        stories = append(stories, Story {
            ID: id,
            Title: title,
            Description: description,
            StartTime: startTime,
            Creator: creatorName,
            IsStoryOwner: isStoryOwner,
        })
    }

    tmpl := template.Must(template.ParseFiles("app/templates/story-list.html", "app/templates/story-list-element.html", "app/templates/spinner.html"))
    tmpl.Execute(w, StoryListData{ Stories: stories, IsUserLoggedIn: sessionErr == nil })
}

type CreateStoryPageData struct {
    StoryID int64
    Title string
    StartTime string
    Description string
    Tasks []Task
}

func CreateStoryPage (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        http.Error(w, "Cannot find valid session", 401)
        return
    }

    _, err = db.Exec("DELETE FROM assignment WHERE task_id IN (SELECT task.id FROM task JOIN story ON story.id = task.story_id AND story.creator_id = $1 AND status = 0)", userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error cleaning up draft stories: %s", err), 500)
        return
    }
    _, err = db.Exec("DELETE FROM task WHERE story_id IN (SELECT story.id FROM story WHERE creator_id = $1 AND status = 0)", userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error cleaning up draft stories: %s", err), 500)
        return
    }
    _, err = db.Exec("DELETE FROM story WHERE creator_id = $1 AND status = 0", userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error cleaning up draft stories: %s", err), 500)
        return
    }

    result, err := db.Exec("INSERT INTO story (creator_id, status) VALUES($1, $2)", userID, 0)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating story draft: %s", err), 500)
        return
    }

    storyID, err := result.LastInsertId()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating story draft: %s", err), 500)
        return
    }

    tasks, err := GetTasks(db)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting task: %s", err), 500)
        return
    }

    tmpl := template.Must(template.ParseFiles("app/templates/create-story.html", "app/templates/spinner.html"))
    err = tmpl.Execute(w, CreateStoryPageData { StoryID: storyID, Tasks: tasks })
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func createTaskToStoryHandler (r *http.Request) (Task, string, int) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        return Task{}, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400
    }
    name := r.PostFormValue("name")
    description := r.PostFormValue("description")
    slots, err := strconv.ParseInt(r.PostFormValue("slots"), 10, 64)
    if err != nil {
        return Task{}, fmt.Sprintf("Cannot parse value %s as integer: %s", r.PostFormValue("slots"), err), 400
    }

    db, err := OpenDB()
    if err != nil {
        return Task{}, fmt.Sprintf("Error connecting to database: %s", err), 500
    }
    defer db.Close()
    _, _, err = auth.ValidateSession(db, r);
    if err != nil {
        return Task{}, "Cannot find valid session", 401
    }

    result, err := db.Exec("INSERT INTO task (story_id, name, description, slots) VALUES($1, $2, $3, $4)", storyID, name, description, slots)
    if err != nil {
        return Task{}, fmt.Sprintf("Error creating task: %s", err), 500
    }

    id, err := result.LastInsertId()
    if err != nil {
        return Task{}, fmt.Sprintf("Error creating task: %s", err), 500
    }

    return  Task{
        ID: id,
        Name: name,
        Description: description,
        SlotsTotal: slots,
        SlotsAssigned: 0,
        AssignmentList: []Assignments{},
        HasJoined: false,
        IsStoryOwner: true,
        IsUserLoggedIn: true,
    }, "", 0
}

func AddTaskToStoryFinalizeHandler (w http.ResponseWriter, r *http.Request) {
    task, errorMsg, errorCode := createTaskToStoryHandler(r)
    if errorCode != 0 {
        http.Error(w, errorMsg, errorCode)
    }

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html", "app/templates/spinner.html"))
    err := tmpl.ExecuteTemplate(w, "task-list-element-base", task)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func AddTaskToStoryHandler (w http.ResponseWriter, r *http.Request) {
    task, errorMsg, errorCode := createTaskToStoryHandler(r)
    if errorCode != 0 {
        http.Error(w, errorMsg, errorCode)
    }

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err := tmpl.ExecuteTemplate(w, "task-list-element-view.html", task)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func ChangeStoryTaskAssignmentHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }
    action := r.PostFormValue("action")

    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        http.Error(w, "Cannot find valid session", 401)
        return
    }

    var result sql.Result
    if action == "join" {
        result, err = db.Exec("INSERT INTO assignment (task_id, assignee_id) VALUES($1, $2)", taskID, userID)
    } else {
        result, err = db.Exec("DELETE FROM assignment WHERE task_id = $1 AND assignee_id = $2", taskID, userID)
    }
    if err != nil {
        http.Error(w, fmt.Sprintf("Error changing task assignment: %s", err), 500)
        return
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error changing task assignment: %s", err), 500)
        return
    }
    if rowsAffected != 1 {
        http.Error(w, "Error changing task assignment: incorrect number of rows changes", 500)
        return
    }

    task, err := GetSingleTask(db, taskID, userID)
    task.IsUserLoggedIn = true

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-list-element-view.html", task)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func ChangeStoryTaskViewHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }

    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        http.Error(w, "Cannot find valid session", 401)
        return
    }

    task, err := GetSingleTask(db, taskID, userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting task data: %s", err), 500)
        return
    }
    task.IsUserLoggedIn = true

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-detail-edit", task)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func TaskDetailHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);
    task, err := GetSingleTask(db, taskID, userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting task data: %s", err), 500)
        return
    }
    task.IsUserLoggedIn = sessionErr != nil

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-detail-view", task)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func ChangeTaskHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    name := r.PostFormValue("name")
    description := r.PostFormValue("description")
    slotsTotal, err := strconv.ParseInt(r.PostFormValue("slots"), 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", r.PostFormValue("slots"), err), 400)
        return
    }
    userID, _, sessionErr := auth.ValidateSession(db, r);
    if sessionErr != nil {
        http.Error(w, "Cannot find valid session", 401)
        return
    }

    result, err := db.Exec(
        "UPDATE task SET name = $1, description = $2, slots = $3 WHERE id = $4",
        name, description, slotsTotal, taskID,
    )
    if err != nil {
        http.Error(w, fmt.Sprintf("Error updating task data: %s", err), 500)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error updating task data: %s", err), 500)
        return
    }
    if rowsAffected != 1 {
        http.Error(w, fmt.Sprintf("Error updating task, incorrect numbers of rows affected: %d", rowsAffected), 500)
        return
    }

    task, err := GetSingleTask(db, taskID, userID)
    task.IsUserLoggedIn = true
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting task data: %s", err), 500)
        return
    }

    slotsToDelete := task.SlotsAssigned - task.SlotsTotal
    if slotsToDelete > 0 {
        result, err := db.Exec(`
            DELETE FROM assignment
            WHERE task_id = $1
            AND id NOT IN
                (SELECT id FROM assignment WHERE task_id = $1 ORDER BY id ASC LIMIT $2)
            `,
            taskID,
            slotsToDelete,
        )
        if err != nil {
            http.Error(w, fmt.Sprintf("Error updating tasks slots: %s", err), 500)
            return
        }

        rowsAffected, err := result.RowsAffected()
        if err != nil {
            http.Error(w, fmt.Sprintf("Error updating tasks slots: %s", err), 500)
            return
        }
        if rowsAffected != slotsToDelete {
            http.Error(w, fmt.Sprintf("Error deleting assignments due to the slots change in task: deleted %d instead of %d", rowsAffected, slotsToDelete), 500)
            return
        }

        assignments, hasJoined, err := GetTaskAssignments(db, taskID, userID)
        if err != nil {
            http.Error(w, fmt.Sprintf("Error getting assignments: %s", err), 500)
            return
        }
        task.HasJoined = hasJoined
        task.AssignmentList = assignments
        task.SlotsAssigned = int64(len(assignments))
    }

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-list-element-view.html", task)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func DeleteStoryTaskHandler (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400)
        return
    }

    result, err := db.Exec("DELETE FROM task WHERE id = $1", id)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error deleting task: %s", err), 500)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error deleting task: %s", err), 500)
        return
    }
    if rowsAffected != 1 {
        http.Error(w, fmt.Sprintf("Error deleting story task, incorrect numbers of rows affected: %d", err), 500)
        return
    }
}

func updateStoryHandler (r *http.Request) (Story, string, int) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        return Story{}, fmt.Sprintf("Cannot parse value %s as integer: %s", vars["id"], err), 400
    }
    db, err := OpenDB()
    if err != nil {
        return Story{}, fmt.Sprintf("Error connecting to database: %s", err), 500

    }
    defer db.Close()
    title := r.PostFormValue("title")
    description := r.PostFormValue("description")
    startTime, err := strconv.ParseInt(r.PostFormValue("time"), 10, 64)
    if err != nil {
        return Story{}, fmt.Sprintf("Cannot parse value %s as integer: %s", r.PostFormValue("time"), err), 400
    }
    userID, userName, sessionErr := auth.ValidateSession(db, r)
    if sessionErr != nil {
        return Story{}, "Cannot find valid session", 401
    }

    result, err := db.Exec(
        "UPDATE story SET title = $1, description = $2, start_time = $3, status = 1 WHERE id = $4 AND creator_id = $5",
        title, description, startTime, storyID, userID,
    )
    if err != nil {
        return Story{}, fmt.Sprintf("Error updating story: %s", err), 500
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return Story{}, fmt.Sprintf("Error updating story: %s", err), 500
    }
    if rowsAffected != 1 {
        return Story{}, fmt.Sprintf("Error updating story, incorrect numbers of rows affected: %d", rowsAffected), 500
    }
    if err != nil {
        return Story{}, fmt.Sprintf("Error updating story: %s", err), 500
    }

    return Story{
        ID: storyID,
        Title: title,
        Description: description,
        StartTime: time.Unix(startTime, 0).Format("02. 01. 2006 15:04"),
        Creator: userName,
        IsStoryOwner: true,
    }, "", 0
}

type StoryViewPageData struct {
    Story Story
}

func ChangeStoryHandler (w http.ResponseWriter, r *http.Request) {
    story, errorString, errorCode := updateStoryHandler(r)
    if errorCode != 0 {
        http.Error(w, errorString, errorCode)
        return
    }

    tmpl := template.Must(template.ParseFiles("app/templates/story-detail.html", "app/templates/spinner.html"))
    err := tmpl.ExecuteTemplate(w, "story-detail-view", StoryViewPageData { Story: story })
    if err != nil {
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}

func FinalizeCreateStoryHandler (w http.ResponseWriter, r *http.Request) {
    _, errorString, errorCode := updateStoryHandler(r)
    if errorCode != 0 {
        http.Error(w, errorString, errorCode)
        return
    }

    w.Header().Add("HX-Trigger", "reload-stories")
}

func DeleteStoryHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        http.Error(w, "Cannot find valid session", 401)
        return
    }

    result, err := db.Exec("DELETE FROM story WHERE id = $1 and creator_id = $2", id, userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error deleting story: %s", err), 500)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error deleting story: %s", err), 500)
        return
    }
    if rowsAffected != 1 {
        http.Error(w, fmt.Sprintf("Error deleting story, incorrect number of rows affected: %d", rowsAffected), 500)
        return
    }
    w.Header().Add("HX-Redirect", "/")
}