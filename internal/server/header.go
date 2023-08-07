package server

import (
    "html/template"
    "log"
    "net/http"
    "zmtwc/sk/internal/auth"
)

func HeaderHandler (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    _, _, err = auth.ValidateSession(db, r);

    tmpl := template.Must(template.ParseFiles("app/templates/header.html"))
    if err == nil {
        tmpl.ExecuteTemplate(w, "logged-in-header", nil)
    } else {
        tmpl.ExecuteTemplate(w, "logged-out-header", nil)
    }
}
