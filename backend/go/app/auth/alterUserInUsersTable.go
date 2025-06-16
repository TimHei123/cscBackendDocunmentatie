package auth

// AlterUserInUsersTable updates user information in the users table with the provided details
func AlterUserInUsersTable(userSid string, email string, studentID int, homeIP string) (map[string]string, error) {

	email = sanitizeInput(email)
	homeIP = sanitizeInput(homeIP)

	db, err := ConnectToDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	stmt, err := db.Prepare("UPDATE users SET email = ?, student_id = ?, home_ip = ? WHERE user_sid = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.Exec(email, studentID, homeIP, userSid)
	if err != nil {
		return nil, err
	}
	return map[string]string{"message": "User updated successfully"}, nil

}
