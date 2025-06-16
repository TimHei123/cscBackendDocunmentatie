package vmware

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func getEnvVar(varName string) string {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file", err)
	}

	var envVar = os.Getenv(varName)

	return envVar
}

func getBoolEnvVar(varname string) bool {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	if os.Getenv(varname) == "true" {
		return true
	} else {
		return false
	}
}
