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

func GetStoryTasks (db *sql.DB, storyID int64, userID int64, isStoryOwner bool, isUserLoggedIn bool) ([]Task, error) {
    tasks := []Task{}
    rows, err := db.Query(`
        SELECT
            story_task.id,
            story_task.assignee_id,
            user.username,
            story_task.task_id,
            task.name
        FROM story_task
        JOIN task on story_task.task_id = task.id
        LEFT JOIN user on story_task.assignee_id = user.id
        WHERE story_task.story_id = $1
        `,
        storyID,
    )
    if err != nil {
        return []Task{}, err
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var assigneeIDOption sql.NullInt64
        var assigneeNameOption sql.NullString
        var taskID int64
        var taskName string

        err = rows.Scan(&id, &assigneeIDOption, &assigneeNameOption, &taskID, &taskName)
        if err != nil {
            return []Task{}, err
        }

        var assigneeID int64
        if assigneeIDOption.Valid {
            assigneeID = assigneeIDOption.Int64
        } else {
            assigneeID = 0
        }
        assigneeName := ""
        if assigneeNameOption.Valid {
            assigneeName = assigneeNameOption.String
        }

        tasks = append(tasks, Task {
            ID: id,
            AssigneeID: assigneeID,
            AssigneeName: assigneeName,
            TaskID: taskID,
            Name: taskName,
            IsStoryOwner: isStoryOwner,
            HasJoined: assigneeID == userID,
            IsUserLoggedIn: isUserLoggedIn,
        })
    }

    return tasks, nil
}

func StoryDetailHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyID, err := strconv.ParseInt(vars["id"], 10, 64)
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);
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

    err = row.Scan(&id, &title, &creatorName, &creatorID, &descriptionOption, &startTimeOption)
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
    isUserLoggedIn := sessionErr == nil
    tasks, err := GetStoryTasks(db, id, userID, isStoryOwner, isUserLoggedIn)

    tmpl := template.Must(template.ParseFiles("app/templates/story-detail.html", "app/templates/task-list-element-view.html", "app/templates/task-list-element.html", "app/templates/spinner.html"))
    err = tmpl.Execute(w, StoryDetail {
        IsUserLoggedIn: isUserLoggedIn,
        Story: Story {
            ID: id,
            Title: title,
            Description: description,
            StartTime: startTime,
            Creator: creatorName,
            IsStoryOwner: isStoryOwner,
        },
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

type Task struct {
    IsUserLoggedIn bool
    IsStoryOwner bool
    HasJoined bool
    ID int64
    TaskID int64
    Name string
    AssigneeID int64
    AssigneeName string
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

    //TODO: reconsider this
    _, err = db.Exec("DELETE FROM story_task WHERE story_id IN (SELECT story.id FROM story WHERE creator_id = $1 AND status = 0)", userID)
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
    taskID := r.PostFormValue("task")

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

    result, err := db.Exec("INSERT INTO story_task (story_id, task_id) VALUES($1, $2)", storyID, taskID)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    row := db.QueryRow("SELECT name FROM Task WHERE task.id = $1", taskID)
    var taskName string
    err = row.Scan(&taskName)
    if err != nil {
        log.Fatal(err)
    }

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html"))
    template.Must(tmpl.New("spinner").ParseFiles("app/templates/spinner.html"))

    err = tmpl.ExecuteTemplate(w, "task-list-element-base", Task{ID: id, Name: taskName, AssigneeID: 0, AssigneeName: "", IsStoryOwner: true, HasJoined: false })
    if err != nil {
        log.Fatal(err)
    }
}

type StoryTaskButtonData struct {
    ID int64
    AssigneeID int64
    AssigneeName string
    HasJoined bool
}

func ChangeStoryTaskHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    storyTaskID, err := strconv.ParseInt(vars["id"], 10, 64)
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
    userID, userName, err := auth.ValidateSession(db, r);
    if err != nil {
        log.Fatal(err)
    }

    var result sql.Result
    var storyTaskData StoryTaskButtonData
    if action == "join" {
        result, err = db.Exec("UPDATE story_task SET assignee_id = $1 WHERE id = $2 AND assignee_id IS NULL", userID, storyTaskID)
        storyTaskData = StoryTaskButtonData {
            ID: storyTaskID,
            AssigneeID: userID,
            AssigneeName: userName,
            HasJoined: true,
        }
    } else {
        result, err = db.Exec("UPDATE story_task SET assignee_id = NULL WHERE id = $1 AND assignee_id = $2", storyTaskID, userID)
        storyTaskData = StoryTaskButtonData {
            ID: storyTaskID,
            AssigneeID: 0,
            AssigneeName: "",
            HasJoined: false,
        }
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

    tmpl := template.Must(template.ParseFiles("app/templates/task-list-element.html", "app/templates/task-list-element-view.html"))

    err = tmpl.ExecuteTemplate(w, "template-controls", storyTaskData)
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

    result, err := db.Exec("DELETE FROM story_task WHERE id = $1", id)
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

func FinalizeCreateStoryHandler (w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
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
    time := r.PostFormValue("time")

    result, err := db.Exec(
        "UPDATE story SET title = $1, description = $2, start_time = $3, status = 1 WHERE id = $4 AND creator_id = $5",
        title, description, time, id, userID,
    )
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        log.Fatal(err)
    }
    if rowsAffected != 1 {
        log.Fatal(errors.New("Error updating entry"))
    }

    w.Header().Add("HX-Trigger-After-Settle", "reload-stories")
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