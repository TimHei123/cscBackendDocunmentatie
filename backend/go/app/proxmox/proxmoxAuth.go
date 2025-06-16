package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

// AuthResponse holds authentication data
type AuthResponse struct {
	Data struct {
		Ticket string `json:"ticket"`
		CSRF   string `json:"CSRFPreventionToken"`
	} `json:"data"`
	Errors string `json:"errors"`
}

// ConnectToServer authenticates with Proxmox and returns the cookie
func ConnectToServer() (string, string, error) {
	serverURL := "https://172.16.1.81:8006"

	err := godotenv.Load()
	if err != nil {
		return "", "", fmt.Errorf("error loading .env file: %v", err)
	}

	username := os.Getenv("PVE_USERNAME")
	password := os.Getenv("PVE_PASSWORD")
	if username == "" || password == "" {
		return "", "", fmt.Errorf("missing PVE_USERNAME or PVE_PASSWORD in environment variables")
	}

	restyClient := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	resp, err := restyClient.R().
		SetFormData(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(serverURL + "/api2/json/access/ticket")
	if err != nil {
		return "", "", fmt.Errorf("failed to authenticate: %v", err)
	}

	var authData AuthResponse
	if err := json.Unmarshal(resp.Body(), &authData); err != nil {
		return "", "", fmt.Errorf("failed to parse authentication response: %v", err)
	}

	if authData.Data.Ticket == "" {
		if authData.Errors != "" {
			return "", "", fmt.Errorf("authentication failed: %s", authData.Errors)
		}
		return "", "", fmt.Errorf("authentication failed: no ticket received")
	}

	authCookie := fmt.Sprintf("PVEAuthCookie=%s", authData.Data.Ticket)
	return authCookie, authData.Data.CSRF, nil
}
