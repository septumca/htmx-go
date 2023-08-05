package server

import (
    "database/sql"
    _ "modernc.org/sqlite"
)


func OpenDB() (*sql.DB, error) {
    return sql.Open("sqlite", "local.db")
}