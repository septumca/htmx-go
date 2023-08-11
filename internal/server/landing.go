package server

import (
    "html/template"
    "log"
    "fmt"
    "os"
    "net/url"
    "net/http"
    "zmtwc/sk/internal/auth"
    "io"
    "encoding/json"
)

type User struct {
    ID int64
    Username string
}

/*
{
  "success": true|false,
  "challenge_ts": timestamp,  // timestamp of the challenge load (ISO format yyyy-MM-dd'T'HH:mm:ssZZ)
  "hostname": string,         // the hostname of the site where the reCAPTCHA was solved
  "error-codes": [...]        // optional
}
*/
type RecaptchaResponse struct {
    Success bool `json:"success"`
    ChallengeTs string `json:"challenge_ts"`
    Hostname string `json:"hostname"`
}

func RegisterPageHandler (w http.ResponseWriter, r *http.Request) {
    recaptchaKey := os.Getenv("RECAPTCHA_CLIENT_KEY")
    tmpl := template.Must(template.ParseFiles("app/templates/register.html", "app/templates/spinner.html"))
    tmpl.Execute(w, recaptchaKey)
}

func DoRegisterHandler (w http.ResponseWriter, r *http.Request) {
    username := r.PostFormValue("username")
    password := r.PostFormValue("password")
    recaptchaResponse := r.PostFormValue("g-recaptcha-response")
    recaptchaKey := os.Getenv("RECAPTCHA_SERVER_KEY")

    if recaptchaResponse == "" {
        http.Error(w, "Please fill out recaptcha", 401)
        return
    }

    resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{"secret": {recaptchaKey}, "response": {recaptchaResponse}})
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting recaptcha response: %s", err), 500)
        return
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting recaptcha body: %s", err), 500)
        return
    }

    var result RecaptchaResponse
    if err := json.Unmarshal(body, &result); err != nil {  // Parse []byte to the go struct pointer
        http.Error(w, fmt.Sprintf("Error parsing recaptcha json from body %s: %s", body, err), 500)
        return
    }

    if result.Success == false {  // Parse []byte to the go struct pointer
        http.Error(w, fmt.Sprintf("Invalid recaptcha: %v", result), 401)
        return
    }

    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    userID, err := auth.SavePasswordForUser(db, username, password)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error storing password: %s", err), 500)
        return
    }

    sessionID, err := auth.GenerateSessionID(db, userID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error generating session ID: %s", err), 500)
        return
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
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    userID, err := auth.IsPasswordMatching(db, username, password)
    if err != nil {
        http.Error(w, "Incorrect password", 401)
        return
    }

    sessionID, err := auth.GenerateSessionID(db, userID)
    if err == nil {
        w.Header().Add("HX-Redirect", "/")
        w.Header().Add("Set-Cookie", "session-id:"+sessionID)
        tmpl := template.Must(template.ParseFiles("app/templates/header.html"))
        tmpl.ExecuteTemplate(w, "logged-in-header", nil)
    } else {
        log.Println(err)
        http.Error(w, fmt.Sprintf("Error generating session ID: %s", err), 500)
    }
}

func DoLogoutHandler (w http.ResponseWriter, r *http.Request) {
    db, err := OpenDB()
    if err != nil {
        http.Error(w, fmt.Sprintf("Error connecting to database: %s", err), 500)
        return
    }
    defer db.Close()

    _ = auth.Logout(db, r)
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
        http.Error(w, fmt.Sprintf("Error building template: %s", err), 500)
    }
}
