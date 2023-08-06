package auth

import (
    "net/http"
    "time"
    "golang.org/x/crypto/bcrypt"
    "database/sql"
    "errors"
    "strings"
    "github.com/google/uuid"
)

func IsSessionValid(db *sql.DB, r *http.Request) (int64, error) {
    cookieHeader := r.Header.Get("Cookie")

    for _, c := range strings.Split(cookieHeader, ",") {
        cookieVals := strings.Split(c, ":")
        if cookieVals[0] == "session-id" {
            row := db.QueryRow("SELECT user, valid_to FROM access_token WHERE token = $1", cookieVals[1])
            var validTo int64
            var userID int64
            err := row.Scan(&userID, &validTo)
            if err != nil {
                return 0, err
            }
            if time.Now().Unix() > validTo {
                return 0, errors.New("Token no longer valid")
            }
            return userID, nil
        }
    }
    return 0, errors.New("Cannot find session-id token")
}

func SavePasswordForUser(db *sql.DB, username string, password string) (int64, error) {
    generatedHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
    if err != nil {
        return 0, err
    }

    result, err := db.Exec("INSERT INTO user (username, password) VALUES($1, $2)", username, generatedHash)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        return 0, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }

    return id, nil
}

func GenerateSessionID(db *sql.DB, userID int64) (string, error) {
    const TokenValidSeconds = 28800;
    sessionID := uuid.New().String()
    _, err := db.Exec("DELETE FROM access_token WHERE user = $1", userID)
    result, err := db.Exec("INSERT INTO access_token (user, token, valid_to) VALUES($1, $2, $3)", userID, sessionID, time.Now().Unix() + TokenValidSeconds)
    if err != nil {
        // http.Error(w, http.StatusText(500), 500)
        return "", err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return "", err
    }
    if rowsAffected != 1 {
        return "", errors.New("Error saving session ID")
    }

    return sessionID, nil
}

func IsPasswordMatching(db *sql.DB, username string, password string) (int64, error) {
    row := db.QueryRow("SELECT user.id, user.password FROM User WHERE user.username = $1", username)
    var dbPassword string
    var userID int64
    err := row.Scan(&userID, &dbPassword)
    if err != nil {
        return 0, err
    }

    err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
    if err != nil {
        return 0, err
    }

    return userID, nil
}