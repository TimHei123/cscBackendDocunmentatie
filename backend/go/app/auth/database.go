package auth

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" // MySQL driver for database/sql
	"github.com/joho/godotenv"
)

var (
	dbUser string
	dbPass string
	dbHost string
	dbPort string
	dbName string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	dbUser = GetEnvVar("DB_USER")
	dbPass = GetEnvVar("DB_PASS")
	dbHost = GetEnvVar("DB_HOST")
	dbPort = GetEnvVar("DB_PORT")
	dbName = GetEnvVar("DB_NAME")
}

// ConnectToDB establishes a connection to the MySQL database and returns a database handle
func ConnectToDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName))
	if err != nil {
		return nil, err
	}

	// ping the database to check if the connection is successful
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Set maximum number of connections
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(20)

	return db, nil
}
