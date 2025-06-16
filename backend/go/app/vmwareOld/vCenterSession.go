package vmware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// GetVCenterSession gets the session ID from vCenter
func GetVCenterSession() string {
	var sessionID string

	// Check if the session ID is already stored in Redis so we don't have to get a new one every time
	sessionID = getFromRedis("session")
	if sessionID != "" && sessionID != "Unauthorized" {
		log.Println("Session ID in Cache: ", sessionID)
		expired := checkIfVCenterSessionIsExpired(sessionID)

		if !expired {
			return sessionID
		}

		log.Println("Session ID in Cache is expired, refreshing session ID")
	} else {
		log.Println("Session ID not found in Cache, refreshing session ID")
	}

	sessionID = refreshVCenterSession()

	log.Println("Session ID from vCenter: ", sessionID)

	return sessionID
}

func getVCenterSession() string {
	defer timeTrack(time.Now(), "getVCenterSession")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	reqBody := map[string]string{
		"username": getEnvVar("VCENTER_USERNAME"),
		"password": getEnvVar("VCENTER_PASSWORD"),
	}

	jsonReqBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("Error marshalling request body: ", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/session", bytes.NewBuffer(jsonReqBody))
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response: ", err)
	}

	defer resp.Body.Close()
	return string(body[1 : len(body)-1])
}

// vCenterFetchSession retrieves a new session ID from vCenter
func vCenterFetchSession() string {
	defer timeTrack(time.Now(), "vCenterFetchSession")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	reqBody := map[string]string{
		"username": getEnvVar("VCENTER_USERNAME"),
		"password": getEnvVar("VCENTER_PASSWORD"),
	}

	jsonReqBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("Error marshalling request body: ", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/session", bytes.NewBuffer(jsonReqBody))
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response: ", err)
	}

	defer resp.Body.Close()
	return string(body[1 : len(body)-1])
}

// checkIfVCenterSessionIsExpired verifies if a vCenter session is still valid
func checkIfVCenterSessionIsExpired(session string) bool {
	defer timeTrack(time.Now(), "checkIfVCenterSessionIsExpired")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("GET", baseURL+"/api/session", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	if resp.StatusCode != 200 {
		return true
	}

	return false
}

// refreshVCenterSession refreshes the vCenter session ID
func refreshVCenterSession() string {
	defer timeTrack(time.Now(), "refreshVCenterSession")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	reqBody := map[string]string{
		"username": getEnvVar("VCENTER_USERNAME"),
		"password": getEnvVar("VCENTER_PASSWORD"),
	}

	jsonReqBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("Error marshalling request body: ", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/session", bytes.NewBuffer(jsonReqBody))
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response: ", err)
	}

	defer resp.Body.Close()
	return string(body[1 : len(body)-1])
}
