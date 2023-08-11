package server

import (
    "html/template"
    "net/http"
    "zmtwc/sk/internal/auth"
    "fmt"
)

func HeaderHandler (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
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
