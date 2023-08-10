package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetSessionID(cookieHeader string) (string, error) {
    for _, c := range strings.Split(cookieHeader, ";") {
        cookieVals := strings.Split(c, ":")
        if strings.TrimSpace(cookieVals[0]) == "session-id" {
            return strings.TrimSpace(cookieVals[1]), nil
        }
    }
    return "", errors.New("Cannot find session-id token")
}

func GetSessionUser(db *sql.DB, sessionID string) (int64, string, error) {
    row := db.QueryRow(`
        SELECT
            access_token.user_id,
            user.username,
            access_token.valid_to
        FROM access_token
        JOIN user ON user.id = access_token.user_id
        WHERE access_token.token = $1`,
        sessionID,
    )
    var validTo int64
    var userID int64
    var userName string
    err := row.Scan(&userID, &userName, &validTo)
    if err != nil {
        return 0, "", err
    }
    if time.Now().Unix() > validTo {
        return 0, "", errors.New("Token no longer valid")
    }
    return userID, userName, nil
}

func ValidateSession(db *sql.DB, r *http.Request) (int64, string, error) {
    cookieHeader := r.Header.Get("Cookie")

    sessionID, err := GetSessionID(cookieHeader)
    if err != nil {
        return 0, "", err
    }
    return GetSessionUser(db, sessionID)
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
    _, err := db.Exec("DELETE FROM access_token WHERE user_id = $1", userID)
    result, err := db.Exec("INSERT INTO access_token (user_id, token, valid_to) VALUES($1, $2, $3)", userID, sessionID, time.Now().Unix() + TokenValidSeconds)
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

func Logout(db *sql.DB, r *http.Request) error {
    userID, _, err := ValidateSession(db, r)

    if err != nil {
        return err;
    }
    result, err := db.Exec("DELETE FROM access_token WHERE user_id = $1", userID)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected != 1 {
        return errors.New("Error clearing up session ID")
    }

    return nil
}