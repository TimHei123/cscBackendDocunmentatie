package auth

import (
	"fmt"
	"strings"
)

// CreateUser creates a new user in the database
func CreateUser(username string, userSid string) (map[string]string, error) {
	fmt.Println("Creating user in users table")

	// Sanitize input
	username = sanitizeInput(username)

	db, err := ConnectToDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Check if user exists
	userRows, err := db.Query("SELECT user_sid FROM users WHERE user_sid = ?", userSid)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()

	if userRows.Next() {
		return map[string]string{"message": "User already exists"}, nil
	}

	// Insert new user
	stmt, err := db.Prepare("INSERT INTO users (username, user_sid) VALUES (?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, userSid)
	if err != nil {
		return nil, err
	}

	return map[string]string{"message": "User created successfully"}, nil
}

// sanitizeInput removes potentially harmful characters to prevent XSS
func sanitizeInput(input string) string {
	// Example: Replace < and > to prevent HTML injection
	input = strings.ReplaceAll(input, "<", "")
	input = strings.ReplaceAll(input, ">", "")
	return input
}
