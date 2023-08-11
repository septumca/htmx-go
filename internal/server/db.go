package server

import (
    "os"
    "database/sql"
)

func OpenDB() (*sql.DB, error) {
    return sql.Open("sqlite", os.Getenv("DB_PATH"))
}

func GetTasks(db *sql.DB) ([]Task, error) {
    rows, err := db.Query("SELECT task.id, task.name FROM task")
    if err != nil {
        return []Task{}, err
    }
    defer rows.Close()

    tasks := []Task{}
    for rows.Next() {
        var id int64
        var name string

        err = rows.Scan(&id, &name)
        if err != nil {
            return []Task{}, err
        }

        tasks = append(tasks, Task { ID: id, Name: name })
    }
    return tasks, nil
}