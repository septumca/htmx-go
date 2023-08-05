package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/gorilla/mux"
    "zmtwc/sk/internal/server"
)

func main() {
    fmt.Println("Starting server at port 8000")

    r := mux.NewRouter()
    r.HandleFunc("/", server.LandingPage).Methods("GET")
    r.HandleFunc("/story", server.CreateStoryHandler).Methods("POST")
    r.HandleFunc("/story", server.StoryListHandler).Methods("GET")
    r.HandleFunc("/story/{id}", server.DeleteStoryHandler).Methods("DELETE")
    http.Handle("/", r)

    log.Printf("Starting server")
    log.Fatal(http.ListenAndServe("127.0.0.1:8000", nil))
}
