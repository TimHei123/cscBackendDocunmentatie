package proxmox

import (
	auth "CSC-SelfServiceBackend/app/auth"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

// ConnectToServer authenticates with Proxmox and returns ticket (auth cookie) and CSRF token
func (h Prox) ListVmsUser(vmIDs []int, user_sid string) ([]map[string]interface{}, error) {
	log.Println("Starting ListVmsUser function")

	serverURL := os.Getenv("PROXMOX_SERVER_URL")
	nodeName := os.Getenv("PROXMOX_NODE")

	if serverURL == "" || nodeName == "" {
		err := fmt.Errorf("PROXMOX_SERVER_URL or PROXMOX_NODE environment variables are not set")
		log.Println(err)
		return nil, err
	}

	restyClient := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	authCookie, csrfToken, err := ConnectToServer()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Proxmox server: %v", err)
	}

	db, err := auth.ConnectToDB()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %v", err)
	}
	defer db.Close()

	query := `
        SELECT vm.vmid, vm.description, vm.user_sid, vm.full_name, vm.expires_at, vm.subdomain, vm.selectedOs, ip.ip
        FROM virtual_machines vm
        LEFT JOIN ip_addresses ip ON vm.vmid = ip.virtual_machine_id
        WHERE vm.user_sid = ?;
    `
	rows, err := db.Query(query, user_sid)
	if err != nil {
		return nil, fmt.Errorf("failed to query virtual machines: %v", err)
	}
	defer rows.Close()

	type dbUser struct {
		FullName     string
		UserSID      string
		ExpiresAt    string
		Vmid         string
		Description  string
		Subdomain    string
		SelectedOs   string
		IPAddressStr string
	}

	userMap := make(map[string]dbUser)

	for rows.Next() {
		var user dbUser
		var subdomain sql.NullString
		var selectedOs sql.NullString
		var ip sql.NullString

		err := rows.Scan(&user.Vmid, &user.Description, &user.UserSID, &user.FullName, &user.ExpiresAt, &subdomain, &selectedOs, &ip)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user info: %v", err)
		}

		if subdomain.Valid {
			user.Subdomain = subdomain.String
		} else {
			user.Subdomain = ""
		}

		if selectedOs.Valid {
			user.SelectedOs = selectedOs.String
		} else {
			user.SelectedOs = ""
		}

		if ip.Valid {
			user.IPAddressStr = ip.String
		} else {
			user.IPAddressStr = ""
		}

		userMap[user.Vmid] = user
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	var vms []map[string]interface{}

	authCookie = strings.TrimPrefix(authCookie, "PVEAuthCookie=")

	for _, vmid := range vmIDs {
		vmidStr := strconv.Itoa(vmid)
		user, ok := userMap[vmidStr]
		if !ok {
			log.Printf("VMID %s not found in userMap, skipping", vmidStr)
			continue
		}

		url := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/status/current", serverURL, nodeName, vmid)
		req := restyClient.R().
			SetHeader("Accept", "application/json").
			SetHeader("CSRFPreventionToken", csrfToken).
			SetCookie(&http.Cookie{Name: "PVEAuthCookie", Value: authCookie})

		resp, err := req.Get(url)
		if err != nil {
			log.Printf("Error fetching VM status for VMID %d: %v", vmid, err)
			continue
		}

		if resp.StatusCode() != 200 {
			log.Printf("Non-200 response for VMID %d: %s", vmid, resp.Status())
			continue
		}

		var jsonResponse struct {
			Data struct {
				Name   string  `json:"name"`
				Status string  `json:"status"`
				Cpu    float64 `json:"cpu"`
				Maxcpu int     `json:"maxcpu"`
				Maxmem int64   `json:"maxmem"`
				Mem    int64   `json:"mem"`
				Uptime int     `json:"uptime"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp.Body(), &jsonResponse); err != nil {
			log.Printf("Error parsing JSON for VMID %d: %v", vmid, err)
			continue
		}

		vmData := map[string]interface{}{
			"vmid":        vmid,
			"name":        jsonResponse.Data.Name,
			"cpu":         jsonResponse.Data.Cpu,
			"maxcpu":      jsonResponse.Data.Maxcpu,
			"maxmem":      jsonResponse.Data.Maxmem,
			"mem":         jsonResponse.Data.Mem,
			"uptime":      jsonResponse.Data.Uptime,
			"status":      jsonResponse.Data.Status,
			"description": user.Description,
			"expiresAt":   user.ExpiresAt,
			"subdomain":   user.Subdomain,
			"selectedOs":  user.SelectedOs,
			"ip_address":  user.IPAddressStr,
		}

		vms = append(vms, vmData)
	}

	log.Printf("Returning %d VMs for user %s", len(vms), user_sid)
	return vms, nil
}

