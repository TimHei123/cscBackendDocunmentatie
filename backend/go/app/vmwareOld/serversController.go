package vmware

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
)

// DBServers represents a server record from the database
type DBServers struct {
	ID              int
	UsersID         string
	VcenterID       string
	Name            string
	Description     string
	EndDate         string
	OperatingSystem string
	Storage         int
	Memory          int
	IP              string
}

// VCenterServers represents a server record from vCenter
type VCenterServers struct {
	MemorySizeMiB int    `json:"memory_size_MiB"`
	VM            string `json:"vm"`
	Name          string `json:"name"`
	PowerState    string `json:"power_state"`
	CPUCount      int    `json:"cpu_count"`
}

// PowerStatusReturn represents a server record with power status
type PowerStatusReturn struct {
	ID              int
	UsersID         string
	VcenterID       string
	Name            string
	Description     string
	EndDate         string
	OperatingSystem string
	Storage         int
	Memory          int
	IP              string
	PowerStatus     string
}

// ServerCreationJSONBody represents the request body for server creation
type ServerCreationJSONBody struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	OperatingSystem string    `json:"operating_system"`
	EndDate         string    `json:"end_date"`
	Storage         int       `json:"storage"`
	Memory          int       `json:"memory"`
	HomeIPs         *[]string `json:"home_ips"`
	SubDomain       *string   `json:"sub_domain"`
	DomainZone      *string   `json:"domain_zone"`
}

// StartScript represents a script configuration for server startup
type StartScript struct {
	User             string `json:"user"`
	Password         string `json:"password"`
	ScriptLocation   string `json:"scriptLocation"`
	ScriptExecutable string `json:"scriptExecutable"`
}

// GetServers retrieves server information based on user permissions and optional ID filter
func GetServers(c echo.Context) error {
	id := c.Param("id")
	userID, isAdmin, _, _ := getUserAssociatedWithJWT(c)
	session := getVCenterSession()
	serversFromVCenter := getPowerStatusFromvCenter(session, "")

	db, err := connectToDB()
	if err != nil {
		log.Println("Error connecting to database: ", err)
	}
	defer db.Close()

	rows, err := getServersFromSQL(db, id, userID, isAdmin)
	if err != nil {
		log.Println("Error executing query: ", err)
	}
	defer rows.Close()

	rowsArr, err := getPowerStatusRows(rows, serversFromVCenter)
	if err != nil {
		log.Println("Error scanning row: ", err)
	}

	if id != "" {
		if len(rowsArr) > 0 {
			return c.JSON(http.StatusOK, rowsArr[0])
		} else {
			return c.JSON(http.StatusNotFound, "No servers found for the given ID")
		}
	}

	return c.JSON(http.StatusOK, rowsArr)
}

// getServersFromSQL retrieves server records from the database based on user permissions
func getServersFromSQL(db *sql.DB, id string, user string, admin bool) (*sql.Rows, error) {
	if id != "" && !admin {
		return db.Query("SELECT id, users_id, vcenter_id, name, description, end_date, operating_system, storage, memory, ip FROM virtual_machines WHERE id = ? and users_id = ?", id, user)
	} else if id != "" && admin {
		return db.Query("SELECT id, users_id, vcenter_id, name, description, end_date, operating_system, storage, memory, ip FROM virtual_machines WHERE id = ?", id)
	} else if admin {
		return db.Query("SELECT id, users_id, vcenter_id, name, description, end_date, operating_system, storage, memory, ip FROM virtual_machines")
	} else {
		return db.Query("SELECT id, users_id, vcenter_id, name, description, end_date, operating_system, storage, memory, ip FROM virtual_machines WHERE users_id = ?", user)
	}
}

// getPowerStatusRows processes database rows and adds power status information
func getPowerStatusRows(rows *sql.Rows, serversFromVCenter []VCenterServers) ([]PowerStatusReturn, error) {
	var rowsArr []PowerStatusReturn
	var wg sync.WaitGroup
	rowChan := make(chan PowerStatusReturn)

	for rows.Next() {
		var s PowerStatusReturn
		err := rows.Scan(&s.ID, &s.UsersID, &s.VcenterID, &s.Name, &s.Description, &s.EndDate, &s.OperatingSystem, &s.Storage, &s.Memory, &s.IP)
		if err != nil {
			return nil, err
		}

		wg.Add(1)
		go func(s PowerStatusReturn) {
			defer wg.Done()
			s.PowerStatus = getVCenterPowerState(s.VcenterID, serversFromVCenter)
			rowChan <- s
		}(s)
	}

	go func() {
		wg.Wait()
		close(rowChan)
	}()

	for row := range rowChan {
		rowsArr = append(rowsArr, row)
	}

	return rowsArr, nil
}

// getVCenterPowerState retrieves the power state of a server from vCenter
func getVCenterPowerState(dbID string, vCenterServers []VCenterServers) string {
	for _, server := range vCenterServers {
		if server.VM == dbID {
			return server.PowerState
		}
	}

	return "UNKNOWN"
}

// DeleteServer removes a server and its associated resources
func DeleteServer(c echo.Context) error {
	id := c.Param("id")
	db, err := connectToDB()
	if err != nil {
		log.Println("Error connecting to database: ", err)
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Error converting ID to int")
	}

	// delete all the DNS records for the server
	err = deleteDNSRecordsForServer(idInt, db)
	if err != nil {
		log.Println("Error deleting DNS records for server: ", err)
		return c.JSON(http.StatusBadRequest, "Error deleting DNS records for server")
	}

	// get the vCenter ID from the database
	var (
		vCenterID  string
		serverName string
	)

	userID, isAdmin, _, studentID := getUserAssociatedWithJWT(c)

	if isAdmin {
		err = db.QueryRow("SELECT vcenter_id, name FROM virtual_machines WHERE id = ?", id).Scan(&vCenterID, &serverName)
	} else {
		err = db.QueryRow("SELECT vcenter_id, name FROM virtual_machines WHERE id = ? and users_id = ?", id, userID).Scan(&vCenterID, &serverName)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Can't find server with that ID")
	}

	// Prepare statement for deleting data
	stmt, err := db.Prepare("DELETE FROM virtual_machines WHERE id = ?")
	if err != nil {
		log.Println("Error preparing statement: ", err)
	}

	_, err = stmt.Exec(id)
	if err != nil {
		log.Println("Error executing statement: ", err)
		return c.JSON(http.StatusBadRequest, "Error deleting server from database")
	}

	err = unassignIPfromVM(vCenterID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Error unassigning IP from VM")
	}

	// delete the server from sophos
	err = removeFirewallFromServerInSophos(studentID, serverName)
	if err != nil {
		log.Println("Error removing firewall from sophos: ", err)
		return c.JSON(http.StatusBadRequest, "Error deleting server from sophos")
	}

	// delete the server from vCenter
	session := getVCenterSession()
	status := deletevCenterVM(session, vCenterID)

	if !status {
		return c.JSON(http.StatusBadRequest, "Error deleting server from vCenter")
	}

	return c.JSON(http.StatusCreated, "Server deleted!")
}

// PowerServer controls the power state of a server
func PowerServer(c echo.Context) error {
	id := c.Param("id")
	status := c.Param("status")
	db, err := connectToDB()
	if err != nil {
		log.Println("Error connecting to database: ", err)
		return c.JSON(http.StatusInternalServerError, "error in the program, please try again later")
	}

	userId, isAdmin, _, _ := getUserAssociatedWithJWT(c)

	// get the vCenter ID from the database
	var vCenterID string

	if isAdmin {
		err = db.QueryRow("SELECT vcenter_id FROM virtual_machines WHERE id = ?", id).Scan(&vCenterID)
	} else {
		err = db.QueryRow("SELECT vcenter_id FROM virtual_machines WHERE id = ? and users_id = ?", id, userId).Scan(&vCenterID)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Can't find server with that ID")
	}

	log.Println("vCenterID: ", vCenterID)

	session := getVCenterSession()
	status = strings.ToUpper(status)

	switch status {
	case "ON":
		{
			success := powerOn(session, vCenterID)
			if !success {
				return c.JSON(http.StatusBadRequest, "Error powering on server")
			}
		}
	case "OFF":
		{
			success := powerOff(session, vCenterID)
			if !success {
				return c.JSON(http.StatusBadRequest, "Error powering off server")
			}
		}
	case "FORCE_OFF":
		{
			success := forcePowerOff(session, vCenterID)
			if !success {
				return c.JSON(http.StatusBadRequest, "Error powering off server")
			}
		}
	case "RESET":
		{
			success := reset(session, vCenterID)
			if !success {
				return c.JSON(http.StatusBadRequest, "Error resetting server")
			}
		}
	default:
		return c.JSON(http.StatusBadRequest, "Invalid status")
	}

	return c.JSON(http.StatusCreated, "Server powered "+status)
}

// CreateServer creates a new server with the specified configuration
func CreateServer(c echo.Context) error {
	userID, _, _, studentID := getUserAssociatedWithJWT(c)
	json := new(ServerCreationJSONBody)
	if err := c.Bind(json); err != nil {
		log.Println("Error binding request: ", err)
		return c.JSON(http.StatusBadRequest, "could not bind request")
	}

	db, err := connectToDB()
	if err != nil {
		log.Println("Error connecting to database: ", err)
		return c.JSON(http.StatusInternalServerError, "could not connect to database")
	}

	if checkIfUserAlreadyHasServerWithName(json.Name, userID, db) {
		return c.JSON(http.StatusBadRequest, "Server with that name already exists")
	}

	if countServersByUser(userID, db) >= 2 {
		return c.JSON(http.StatusBadRequest, "You already have 2 servers")
	}

	session := getVCenterSession()
	valid, message, endDate := validateServerCreation(json, session)
	if !valid {
		return c.JSON(http.StatusBadRequest, message)
	}

	err = createServerInDB(userID, json, endDate, db)
	if err != nil {
		log.Println("Error creating server in database: ", err)
		return c.JSON(http.StatusInternalServerError, "could not create server in database")
	}

	var vCenterID string
	err = db.QueryRow("SELECT vcenter_id FROM virtual_machines WHERE name = ? and users_id = ?", json.Name, userID).Scan(&vCenterID)
	if err != nil {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "database", "", db)
		return c.JSON(http.StatusInternalServerError, "could not get vCenter ID from database")
	}

	ip := findEmptyIp()
	if ip == "" {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "ip", "", db)
		return c.JSON(http.StatusInternalServerError, "No IP addresses available")
	}

	err = claimIp(ip)
	if err != nil {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "ip", "", db)
		return c.JSON(http.StatusInternalServerError, "Error claiming IP")
	}

	err = assignIPToVM(ip, vCenterID)
	if err != nil {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "ip", ip, db)
		return c.JSON(http.StatusInternalServerError, "could not assign IP to VM")
	}

	err = updateServerWithVCenterID(vCenterID, json.Name, userID, ip, db)
	if err != nil {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "database", ip, db)
		return c.JSON(http.StatusInternalServerError, "could not update server with vCenter ID")
	}

	err = createFirewallRuleForServerCreation(ip, studentID, json.Name)
	if err != nil {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "firewall", ip, db)
		return c.JSON(http.StatusInternalServerError, "could not create firewall rule")
	}

	err = addUsersToFirewall(studentID, *json)
	if err != nil {
		handleFailedCreation(json.Name, userID, studentID, vCenterID, "firewall", ip, db)
		return c.JSON(http.StatusInternalServerError, "could not add users to firewall")
	}

	return c.JSON(http.StatusCreated, "Server created!")
}

// validateServerCreation validates the server creation request
func validateServerCreation(json *ServerCreationJSONBody, session string) (bool, string, time.Time) {
	if json.Name == "" {
		return false, "Name is required", time.Time{}
	}

	if json.Description == "" {
		return false, "Description is required", time.Time{}
	}

	if json.OperatingSystem == "" {
		return false, "Operating system is required", time.Time{}
	}

	if json.EndDate == "" {
		return false, "End date is required", time.Time{}
	}

	endDate, err := time.Parse("2006-01-02", json.EndDate)
	if err != nil {
		return false, "Invalid end date format", time.Time{}
	}

	if endDate.Before(time.Now()) {
		return false, "End date must be in the future", time.Time{}
	}

	if json.Storage < 1 {
		return false, "Storage must be at least 1", time.Time{}
	}

	if json.Memory < 1 {
		return false, "Memory must be at least 1", time.Time{}
	}

	return true, "", endDate
}

// createServerInDB creates a new server record in the database
func createServerInDB(userID string, json *ServerCreationJSONBody, endDate time.Time, db *sql.DB) error {
	_, err := db.Exec("INSERT INTO virtual_machines (users_id, name, description, end_date, operating_system, storage, memory) VALUES (?, ?, ?, ?, ?, ?, ?)", userID, json.Name, json.Description, endDate, json.OperatingSystem, json.Storage, json.Memory)
	return err
}

// updateServerWithVCenterID updates the server record with vCenter information
func updateServerWithVCenterID(vCenterID, name, userID, ip string, db *sql.DB) error {
	_, err := db.Exec("UPDATE virtual_machines SET vcenter_id = ?, ip = ? WHERE name = ? and users_id = ?", vCenterID, ip, name, userID)
	return err
}

// createFirewallRuleForServerCreation creates firewall rules for a new server
func createFirewallRuleForServerCreation(ip, studentID, serverName string) error {
	return createIPHostInSopohos(ip, studentID, serverName)
}

// addUsersToFirewall adds users to the firewall rules
func addUsersToFirewall(studentID string, json ServerCreationJSONBody) error {
	if json.HomeIPs != nil {
		for _, ip := range *json.HomeIPs {
			err := addIpToSophos(studentID, ip, 0)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// readStartScript reads the start script configuration for a template
func readStartScript(templateName string) (StartScript, error) {
	var script StartScript
	file, err := os.Open("startScripts/" + templateName + ".json")
	if err != nil {
		return script, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&script)
	return script, err
}

// handleFailedCreation handles cleanup when server creation fails
func handleFailedCreation(serverName, userID, studentID, vCenterID, serverCreationStep, ip string, db *sql.DB) {
	log.Println("Server creation failed at step: ", serverCreationStep)
	deleteServerFromDB(serverName, userID, db)
	if vCenterID != "" {
		session := getVCenterSession()
		deletevCenterVM(session, vCenterID)
	}
	if ip != "" {
		unassignIPfromVM(vCenterID)
	}
	if serverCreationStep == "firewall" {
		removeFirewallFromServerInSophos(studentID, serverName)
	}
}

// deleteServerFromDB removes a server record from the database
func deleteServerFromDB(serverName, userID string, db *sql.DB) {
	_, err := db.Exec("DELETE FROM virtual_machines WHERE name = ? and users_id = ?", serverName, userID)
	if err != nil {
		log.Println("Error deleting server from database: ", err)
	}
}

// checkIfUserAlreadyHasServerWithName checks if a user already has a server with the given name
func checkIfUserAlreadyHasServerWithName(name, userID string, db *sql.DB) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM virtual_machines WHERE name = ? and users_id = ?)", name, userID).Scan(&exists)
	if err != nil {
		log.Println("Error checking if user already has server with name: ", err)
		return true
	}
	return exists
}

// checkIfServerExistsInDB checks if a server exists in the database
func checkIfServerExistsInDB(id string, db *sql.DB) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM virtual_machines WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		log.Println("Error checking if server exists in database: ", err)
		return false
	}
	return exists
}

// checkIfServerBelongsToUser checks if a server belongs to a specific user
func checkIfServerBelongsToUser(serverID, userID string, db *sql.DB) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM virtual_machines WHERE id = ? and users_id = ?)", serverID, userID).Scan(&exists)
	if err != nil {
		log.Println("Error checking if server belongs to user: ", err)
		return false
	}
	return exists
}

// countServersByUser counts the number of servers owned by a user
func countServersByUser(userID string, db *sql.DB) int {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM virtual_machines WHERE users_id = ?", userID).Scan(&count)
	if err != nil {
		log.Println("Error counting servers by user: ", err)
		return 0
	}
	return count
}
