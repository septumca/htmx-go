package server

import (
    "log"
    "net/http"
    "html/template"
    "database/sql"

    "github.com/gorilla/mux"
)

type Story struct {
    ID int64
    Title string
    Creator string
}

func StoryListHandler (w http.ResponseWriter, r *http.Request) {
    stories := []Story{}

    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    rows, err := db.Query("SELECT story.id, story.title, user.username FROM story JOIN user on story.creator = user.id")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var title string
        var creator string

        err = rows.Scan(&id, &title, &creator)
        if err != nil {
            log.Fatal(err)
        }

        stories = append(stories, Story { ID: id, Title: title, Creator: creator })
    }

    context := map[string][]Story{
        "Stories": stories,
    }

    // fmt.Printf("context: %v\n", context)

    tmpl := template.Must(template.ParseFiles("templates/story-list.html", "templates/story-list-element.html", "templates/spinner.html"))
    tmpl.Execute(w, context)
}

func CreateStoryHandler (w http.ResponseWriter, r *http.Request) {
    title := r.PostFormValue("title")
    creator := r.PostFormValue("creator")

    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    row := db.QueryRow("SELECT user.username FROM User WHERE user.id = $1", creator)
    var username string
	err = row.Scan(&username)
    if err == sql.ErrNoRows {
		// http.NotFound(w, r)
		log.Fatal(err)
	} else if err != nil {
		// http.Error(w, http.StatusText(500), 500)
		log.Fatal(err)
	}

    result, err := db.Exec("INSERT INTO story (title, creator) VALUES($1, $2)", title, creator)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    tmpl := template.Must(template.New("story-list-element").ParseFiles("templates/story-list-element.html"))
    template.Must(tmpl.New("spinner").ParseFiles("templates/spinner.html"))

    err = tmpl.ExecuteTemplate(w, "story-list-element", Story{ID: id, Title: title, Creator: username})
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

    result, err := db.Exec("DELETE FROM story WHERE id = $1", id)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        log.Fatal(err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil || rowsAffected != 1 {
        log.Fatal(err)
    }
    w.WriteHeader(http.StatusOK)
}