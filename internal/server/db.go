package server

import (
    "database/sql"
)

func OpenDB() (*sql.DB, error) {
    return sql.Open("sqlite", "app/local.db")
}
