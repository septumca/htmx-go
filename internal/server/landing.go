package server

import (
    "html/template"
    "log"
    "net/http"
    "zmtwc/sk/internal/auth"
)

type User struct {
    ID int64
    Username string
}

func RegisterPageHandler (w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("app/templates/register.html", "app/templates/spinner.html"))
    tmpl.Execute(w, nil)
}

func DoRegisterHandler (w http.ResponseWriter, r *http.Request) {
    username := r.PostFormValue("username")
    password := r.PostFormValue("password")
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userID, err := auth.SavePasswordForUser(db, username, password)
    if err != nil {
        log.Fatal(err)
    }

    sessionID, err := auth.GenerateSessionID(db, userID)
    if err != nil {
        log.Println(err)
    }
    w.Header().Add("HX-Redirect", "/")
    w.Header().Add("Set-Cookie", "session-id:"+sessionID)
}

func LoginPageHandler (w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("app/templates/login.html", "app/templates/spinner.html"))
    tmpl.Execute(w, nil)
}

func DoLoginHandler (w http.ResponseWriter, r *http.Request) {
    username := r.PostFormValue("username")
    password := r.PostFormValue("password")
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userID, err := auth.IsPasswordMatching(db, username, password)
    if err != nil {
        log.Fatal(err)
    }

    sessionID, err := auth.GenerateSessionID(db, userID)
    if err == nil {
        w.Header().Add("HX-Redirect", "/")
        w.Header().Add("Set-Cookie", "session-id:"+sessionID)
        tmpl := template.Must(template.ParseFiles("app/templates/header.html"))
        tmpl.ExecuteTemplate(w, "logged-in-header", nil)
    } else {
        log.Println(err)
        http.Error(w, http.StatusText(401), 401)
    }
}

func DoLogoutHandler (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    err = auth.Logout(db, r)
    if err != nil {
        log.Fatal(err)
    }

    w.Header().Add("HX-Redirect", "/")
}

type LandingPageData struct {
    // Users []User
    IsUserLoggedIn bool
}

func LandingPage (w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles(
        "app/templates/index.html",
        "app/templates/create-story.html",
        "app/templates/spinner.html",
    ))
    err := tmpl.Execute(w, nil)
    if err != nil {
        log.Fatal(err)
    }
}
