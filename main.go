package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/gorilla/mux"
    _ "modernc.org/sqlite"

    "zmtwc/sk/internal/server"
)

func main() {
    fmt.Println("Starting server at port 8000")

    r := mux.NewRouter()
    r.HandleFunc("/", server.LandingPage).Methods("GET")
    r.HandleFunc("/login", server.LoginPageHandler).Methods("GET")
    r.HandleFunc("/register", server.RegisterPageHandler).Methods("GET")

    r.HandleFunc("/view/header", server.HeaderHandler).Methods("GET")
    r.HandleFunc("/view/story", server.StoryListHandler).Methods("GET")
    r.HandleFunc("/view/create_story", server.CreateStoryPage).Methods("GET")

    r.HandleFunc("/login", server.DoLoginHandler).Methods("POST")
    r.HandleFunc("/register", server.DoRegisterHandler).Methods("POST")
    r.HandleFunc("/logout", server.DoLogoutHandler).Methods("POST")

    r.HandleFunc("/story/{id}/task", server.AddTaskToStoryHandler).Methods("POST")
    r.HandleFunc("/task/{id}", server.DeleteStoryTaskHandler).Methods("DELETE")
    r.HandleFunc("/task/{id}", server.ChangeStoryTaskHandler).Methods("PUT")
    r.HandleFunc("/story/{id}/finalize", server.FinalizeCreateStoryHandler).Methods("PUT")
    r.HandleFunc("/story/{id}", server.DeleteStoryHandler).Methods("DELETE")
    r.HandleFunc("/story/{id}", server.StoryDetailHandler).Methods("GET")

    http.Handle("/", r)

    log.Printf("Starting server")
    log.Fatal(http.ListenAndServe("127.0.0.1:8000", nil))
}
