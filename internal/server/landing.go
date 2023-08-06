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
    log.Printf("Registration and login successfull")
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
        log.Printf("Login successfull")
        w.Header().Add("HX-Redirect", "/")
        w.Header().Add("Set-Cookie", "session-id:"+sessionID)
    } else {
        log.Println(err)
        http.Error(w, http.StatusText(401), 401)
    }
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

    tmpl := template.Must(template.ParseFiles("app/templates/index.html", "app/templates/spinner.html"))
    tmpl.Execute(w, users)
}