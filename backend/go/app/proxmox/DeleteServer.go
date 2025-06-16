package proxmox

import (
	auth "CSC-SelfServiceBackend/app/auth"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
)

// DeleteServer removes a VM from Proxmox and its associated database record
func (h Prox) DeleteServer(vmID int, userSid string) (string, error) {
	serverURL := os.Getenv("PROXMOX_SERVER_URL")

	// db query
	db, err := auth.ConnectToDB()
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %v", err)
	}
	stmt, err := db.Prepare("DELETE FROM virtual_machines WHERE vmid = ? AND user_sid = ?")
	if err != nil {
		return "", fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(vmID, userSid)
	if err != nil {
		return "", fmt.Errorf("failed to delete VM record from database: %v", err)
	}
	defer db.Close()

	// Create REST client with TLS config
	restyClient := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	// Authenticate
	authCookie, csrfToken, err := ConnectToServer()
	if err != nil {
		return "", fmt.Errorf("authentication error: %v", err)
	}

	// Make the API request to create the serve
	nodeName := os.Getenv("PROXMOX_NODE")

	if nodeName == "" {
		return "", fmt.Errorf("PROXMOX_NODE is not set")
	}

	response, err := restyClient.R().
		SetHeader("Cookie", authCookie).
		SetHeader("CSRFPreventionToken", csrfToken).
		Delete(fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d", serverURL, nodeName, vmID))
	if err != nil {
		return "", fmt.Errorf("failed to delete VM %d: %v", vmID, err)
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("failed to delete VM %d: %s", vmID, response.String())
	}

	stmtDelete, err := db.Prepare("UPDATE `ip_addresses` SET `virtual_machine_id` = NULL WHERE `ip_addresses`.`virtual_machine_id` = ?; ")
	if err != nil {
		return "", fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmtDelete.Close()
	_, err = stmtDelete.Exec(vmID)
	if err != nil {
		return "", fmt.Errorf("failed to alter ip record from database: %v", err)
	}

	return "VM deleted successfully", nil
}
