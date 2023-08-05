package server

import (
    "log"
    "net/http"
    "html/template"
)

type User struct {
    ID int64
    Username string
}

func LandingPage (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    rows, err := db.Query("SELECT user.id, user.username FROM user")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    users := []User{}
    for rows.Next() {
        var id int64
        var username string

        err = rows.Scan(&id, &username)
        if err != nil {
            log.Fatal(err)
        }

        users = append(users, User { ID: id, Username: username })
    }

    tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/spinner.html"))
    tmpl.Execute(w, users)
}