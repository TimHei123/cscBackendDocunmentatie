package auth

import (
	"database/sql"
	"fmt"
)

// GetUserDataFromDB retrieves user data from the database
func GetUserDataFromDB(userSid string) (map[string]string, error) {
	db, err := ConnectToDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Add all the fields you need from the DB
	userRows, err := db.Query(`
		SELECT username, email, student_id, home_ip
		FROM users
		WHERE user_sid = ?
	`, userSid)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()

	if !userRows.Next() {
		return nil, fmt.Errorf("user not found for sid: %s", userSid)
	}

	var (
		username  string
		email     sql.NullString
		studentID sql.NullInt64
		homeIP    sql.NullString
	)

	err = userRows.Scan(&username, &email, &studentID, &homeIP)
	if err != nil {
		return nil, err
	}

	// Use fallback values if NULL
	userData := map[string]string{
		"username":   username,
		"email":      "",
		"student_id": "",
		"home_ip":    "",
	}

	if email.Valid {
		userData["email"] = email.String
	}
	if studentID.Valid {
		userData["student_id"] = fmt.Sprintf("%d", studentID.Int64)
	}
	if homeIP.Valid {
		userData["home_ip"] = homeIP.String
	}

	return userData, nil
}
