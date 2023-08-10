package server

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
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
        log.Fatal(err)
    }
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    _, _, err = auth.ValidateSession(db, r);
    if err != nil {
        log.Fatal(err)
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
        log.Fatal(err)
    }

    description := ""
    if descriptionOption.Valid {
        description = descriptionOption.String
    }
    startTimeString := time.Unix(startTime, 0).Format("2006-01-02T15:04")

    tmpl := template.Must(template.ParseFiles("app/templates/story-detail.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "story-detail-edit", StoryEditPageData {
        ID: id,
        Title: title,
        Description: description,
        StartTime: startTimeString,
    })
    if err != nil {
        log.Fatal(err)
    }
}

type StoryViewPageData struct {
    Story Story
}

func ChangeStoryHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    title := r.PostFormValue("title")
    description := r.PostFormValue("description")
    startTime, err := strconv.ParseInt(r.PostFormValue("time"), 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    userID, userName, sessionErr := auth.ValidateSession(db, r);
    if sessionErr != nil {
        log.Fatal(err)
    }

    err = UpdateStory(db, storyID, userID, title, description, startTime)
    if err != nil {
        log.Fatal(err)
    }

    tmpl := template.Must(template.ParseFiles("app/templates/story-detail.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "story-detail-view", StoryViewPageData { Story: Story{
        ID: storyID,
        Title: title,
        Description: description,
        StartTime: time.Unix(startTime, 0).Format("02. 01. 2006 15:04"),
        Creator: userName,
        IsStoryOwner: true,
    }})
    if err != nil {
        log.Fatal(err)
    }
}

func StoryDetailHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);

    story, err := GetStoryData(db, storyID, userID)
    if err != nil {
        log.Fatal(err)
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
        log.Fatal(err)
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
        log.Fatal(err)
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
        log.Fatal(err)
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
            log.Fatal(err)
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
    Tasks []Task
}

func CreateStoryPage (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        http.Error(w, http.StatusText(401), 401)
        return
    }

    _, err = db.Exec("DELETE FROM assignment WHERE task_id IN (SELECT task.id FROM task JOIN story ON story.id = task.story_id AND story.creator_id = $1 AND status = 0)", userID)
    if err != nil {
        log.Fatal(err)
    }
    _, err = db.Exec("DELETE FROM task WHERE story_id IN (SELECT story.id FROM story WHERE creator_id = $1 AND status = 0)", userID)
    if err != nil {
        log.Fatal(err)
    }
    _, err = db.Exec("DELETE FROM story WHERE creator_id = $1 AND status = 0", userID)
    if err != nil {
        log.Fatal(err)
    }


    result, err := db.Exec("INSERT INTO story (creator_id, status) VALUES($1, $2)", userID, 0)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    storyID, err := result.LastInsertId()
    if err != nil {
        log.Fatal(err)
    }

    tasks, err := GetTasks(db)
    if err != nil {
        log.Fatal(err)
    }

    tmpl := template.Must(template.ParseFiles("app/templates/create-story.html", "app/templates/spinner.html"))
    err = tmpl.Execute(w, CreateStoryPageData { StoryID: storyID, Tasks: tasks })
    if err != nil {
        log.Fatal(err)
    }
}

func AddTaskToStoryHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    name := r.PostFormValue("name")
    description := r.PostFormValue("description")
    slots, err := strconv.ParseInt(r.PostFormValue("slots"), 10, 64)
    if err != nil {
        log.Fatal(err)
    }

    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    defer db.Close()
    _, _, err = auth.ValidateSession(db, r);
    if err != nil {
        log.Fatal(err)
    }

    result, err := db.Exec("INSERT INTO task (story_id, name, description, slots) VALUES($1, $2, $3, $4)", storyID, name, description, slots)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html"))
    template.Must(tmpl.New("spinner").ParseFiles("app/templates/spinner.html"))

    err = tmpl.ExecuteTemplate(w, "task-list-element-base", Task{ID: id, Name: name, Description: description, SlotsTotal: slots, SlotsAssigned: 0 })
    if err != nil {
        log.Fatal(err)
    }
}

func ChangeStoryTaskAssignmentHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    action := r.PostFormValue("action")

    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    defer db.Close()
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        log.Fatal(err)
    }

    var result sql.Result
    if action == "join" {
        result, err = db.Exec("INSERT INTO assignment (task_id, assignee_id) VALUES($1, $2)", taskID, userID)
    } else {
        result, err = db.Exec("DELETE FROM assignment WHERE task_id = $1 AND assignee_id = $2", taskID, userID)
    }
    if err != nil {
        log.Fatal(err)
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }
    if rowsAffected != 1 {
        log.Fatal(fmt.Errorf("Error action: %s task, rows affected: %d", action, rowsAffected))
    }

    task, err := GetSingleTask(db, taskID, userID)
    task.IsUserLoggedIn = true

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-list-element-view.html", task)
    if err != nil {
        log.Fatal(err)
    }
}

func ChangeStoryTaskViewHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }

    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    defer db.Close()
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        log.Fatal(err)
    }

    task, err := GetSingleTask(db, taskID, userID)
    if err != nil {
        log.Fatal(err)
    }
    task.IsUserLoggedIn = true

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-detail-edit", task)
    if err != nil {
        log.Fatal(err)
    }
}

func TaskDetailHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);
    task, err := GetSingleTask(db, taskID, userID)
    if err != nil {
        log.Fatal(err)
    }
    task.IsUserLoggedIn = sessionErr != nil

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-detail-view", task)
    if err != nil {
        log.Fatal(err)
    }
}

func ChangeTaskHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    name := r.PostFormValue("name")
    description := r.PostFormValue("description")
    slotsTotal, err := strconv.ParseInt(r.PostFormValue("slots"), 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    userID, _, sessionErr := auth.ValidateSession(db, r);
    if sessionErr != nil {
        log.Fatal(err)
    }

    result, err := db.Exec(
        "UPDATE task SET name = $1, description = $2, slots = $3 WHERE id = $4",
        name, description, slotsTotal, taskID,
    )
    if err != nil {
        log.Fatal(err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        log.Fatal(err)
    }
    if rowsAffected != 1 {
        log.Fatal(errors.New("Error updating task"))
    }

    task, err := GetSingleTask(db, taskID, userID)
    task.IsUserLoggedIn = true
    if err != nil {
        log.Fatal(err)
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
            log.Fatal(err)
        }

        rowsAffected, err := result.RowsAffected()
        if err != nil {
            log.Fatal(err)
        }
        if rowsAffected != slotsToDelete {
            log.Fatal(fmt.Errorf("Error deleting assignments due to the slots change in task: deleted %d instead of %d", rowsAffected, slotsToDelete))
        }

        assignments, hasJoined, err := GetTaskAssignments(db, taskID, userID)
        if err != nil {
            log.Fatal(err)
        }
        task.HasJoined = hasJoined
        task.AssignmentList = assignments
        task.SlotsAssigned = int64(len(assignments))
    }

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.ExecuteTemplate(w, "task-list-element-view.html", task)
    if err != nil {
        log.Fatal(err)
    }
}

func DeleteStoryTaskHandler (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    defer db.Close()
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }

    result, err := db.Exec("DELETE FROM task WHERE id = $1", id)
    if err != nil {
        log.Fatal(err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        log.Fatal(err)
    }
    if rowsAffected != 1 {
        log.Fatal(fmt.Errorf("Error deleting story task %d", rowsAffected))
    }
}

func UpdateStory(db *sql.DB, storyID int64, userID int64, title string, description string, time int64) error {
    result, err := db.Exec(
        "UPDATE story SET title = $1, description = $2, start_time = $3, status = 1 WHERE id = $4 AND creator_id = $5",
        title, description, time, storyID, userID,
    )
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected != 1 {
        return errors.New("Error updating story")
    }
    return nil
}

func FinalizeCreateStoryHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        log.Fatal(err)
    }
    db, err := OpenDB()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        return
    }
    defer db.Close()
    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        log.Fatal(err)
    }
    title := r.PostFormValue("title")
    description := r.PostFormValue("description")
    time, err := strconv.ParseInt(r.PostFormValue("time"), 10, 64)
    if err != nil {
        log.Fatal(err)
    }

    err = UpdateStory(db, id, userID, title, description, time)
    if err != nil {
        log.Fatal(err)
    }

    w.Header().Add("HX-Trigger", "reload-stories")
}

func DeleteStoryHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userID, _, err := auth.ValidateSession(db, r);
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    result, err := db.Exec("DELETE FROM story WHERE id = $1 and creator_id = $2", id, userID)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        log.Fatal(err)
    }
    if rowsAffected != 1 {
        log.Fatal(errors.New("Error deleting story"))
    }
    w.Header().Add("HX-Redirect", "/")
}