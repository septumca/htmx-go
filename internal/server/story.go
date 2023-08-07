package server

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"zmtwc/sk/internal/auth"

	"github.com/gorilla/mux"
)

type Story struct {
    ID int64
    Title string
    Creator string
    CanBeDeleted bool
}

func StoryListHandler (w http.ResponseWriter, r *http.Request) {
    stories := []Story{}

    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userID, _, sessionErr := auth.ValidateSession(db, r);
    rows, err := db.Query("SELECT story.id, story.title, story.creator, user.username FROM story JOIN user on story.creator = user.id")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var title string
        var creatorName string
        var creatorID int64

        err = rows.Scan(&id, &title, &creatorID, &creatorName)
        if err != nil {
            log.Fatal(err)
        }

        canBeDeleted := sessionErr == nil && userID == creatorID
        stories = append(stories, Story { ID: id, Title: title, Creator: creatorName, CanBeDeleted: canBeDeleted })
    }

    context := map[string][]Story{
        "Stories": stories,
    }

    tmpl := template.Must(template.ParseFiles("app/templates/story-list.html", "app/templates/story-list-element.html", "app/templates/spinner.html"))
    tmpl.Execute(w, context)
}

func CreateStoryHandler (w http.ResponseWriter, r *http.Request) {
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
    title := r.PostFormValue("title")


    result, err := db.Exec("INSERT INTO story (title, creator) VALUES($1, $2)", title, userID)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    tmpl := template.Must(template.New("story-list-element").ParseFiles("app/templates/story-list-element.html"))
    template.Must(tmpl.New("spinner").ParseFiles("app/templates/spinner.html"))

    err = tmpl.ExecuteTemplate(w, "story-list-element", Story{ID: id, Title: title, Creator: userName, CanBeDeleted: true })
    if err != nil {
        log.Fatal(err)
    }
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

    result, err := db.Exec("DELETE FROM story WHERE id = $1 and creator = $2", id, userID)
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
    w.WriteHeader(http.StatusOK)
}