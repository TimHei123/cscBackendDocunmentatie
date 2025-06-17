package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	hypervisors "CSC-SelfServiceBackend/app"
	"CSC-SelfServiceBackend/app/auth"
	proxmox "CSC-SelfServiceBackend/app/proxmox"
	vmware "CSC-SelfServiceBackend/app/vmwareOld"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

// ErrorResponse represents the standard error response structure
type ErrorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
	Code  string `json:"code,omitempty"`
}

// Error codes
const (
	ValidationError = "VALIDATION_ERROR"
	AuthError      = "AUTH_ERROR"
	NotFound       = "NOT_FOUND"
	Conflict       = "CONFLICT"
	ServerError    = "SERVER_ERROR"
)

// Helper function to create error responses
func createErrorResponse(c echo.Context, status int, message string, field string, code string) error {
	err := ErrorResponse{
		Error: message,
		Field: field,
		Code:  code,
	}
	return c.JSON(status, err)
}

// Helper function to validate VM creation parameters
func validateVMCreationParams(requestBody map[string]interface{}) (string, string) {
	// Validate memory
	memory, ok := requestBody["memory"].(float64)
	if !ok || memory < 512 || memory > 16384 {
		return "memory", "Geheugen moet tussen 512MB en 16GB liggen"
	}

	// Validate cores
	cores, ok := requestBody["cores"].(float64)
	if !ok || cores < 1 || cores > 8 {
		return "cores", "CPU cores moet tussen 1 en 8 liggen"
	}

	// Validate disk size
	diskSize, ok := requestBody["DiskSize"].(float64)
	if !ok || diskSize < 10 || diskSize > 500 {
		return "DiskSize", "Schijfgrootte moet tussen 10GB en 500GB liggen"
	}

	// Validate OS
	os, ok := requestBody["os"].(string)
	if !ok {
		return "os", "Ongeldig besturingssysteem"
	}
	validOS := map[string]bool{
		"Ubuntu 22.04":    true,
		"Debian 12":       true,
		"CentOS 9":        true,
		"Windows Server 2022": true,
	}
	if !validOS[os] {
		return "os", "Ongeldig besturingssysteem. Kies uit: Ubuntu 22.04, Debian 12, CentOS 9, Windows Server 2022"
	}

	// Validate name
	name, ok := requestBody["name"].(string)
	if !ok || len(name) < 3 || len(name) > 63 {
		return "name", "Naam moet tussen 3 en 63 karakters lang zijn"
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(name) {
		return "name", "Naam mag alleen letters, cijfers en streepjes bevatten"
	}

	// Validate subdomain
	subdomain, ok := requestBody["subdomain"].(string)
	if !ok || len(subdomain) < 3 || len(subdomain) > 63 {
		return "subdomain", "Subdomein moet tussen 3 en 63 karakters lang zijn"
	}
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(subdomain) {
		return "subdomain", "Subdomein mag alleen kleine letters, cijfers en streepjes bevatten"
	}

	return "", ""
}

func main() {
	var hypervisorMap = map[string]hypervisors.HypervisorFunctions{
		"proxmox": proxmox.Prox{},
	}

	e := echo.New()

	a := e.Group("/auth")
	api := e.Group("/api")
	api.Use(auth.CheckIfLoggedIn)

	a.POST("/login", auth.Login)
	a.POST("/resetRequest", auth.ResetRequest)
	a.POST("/resetPassword", auth.ResetPassword)

	api.POST("/change-user-info", func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return createErrorResponse(c, http.StatusUnauthorized, err.Error(), "", AuthError)
		}

		var requestBody map[string]interface{}
		if err := c.Bind(&requestBody); err != nil {
			return createErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error(), "", ValidationError)
		}

		email, ok := requestBody["email"].(string)
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Invalid email value", "email", ValidationError)
		}
		studentID, ok := requestBody["student_id"].(float64)
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Invalid student_id value", "student_id", ValidationError)
		}
		homeIP, ok := requestBody["home_ip"].(string)
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Invalid home_ip value", "home_ip", ValidationError)
		}

		student_id_int := int(studentID)

		result, err := auth.AlterUserInUsersTable(sid, email, student_id_int, homeIP)
		if err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, err.Error(), "", ServerError)
		}
		return c.JSON(http.StatusOK, result)
	})

	api.GET("/get-user-info", func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return createErrorResponse(c, http.StatusUnauthorized, err.Error(), "", AuthError)
		}

		if sid == "" {
			return createErrorResponse(c, http.StatusBadRequest, "Missing sid header", "sid", ValidationError)
		}

		result, err := auth.GetUserDataFromDB(sid)
		if err != nil {
			c.Logger().Errorf("Error getting user data: %v", err)
			return createErrorResponse(c, http.StatusInternalServerError, err.Error(), "", ServerError)
		}

		return c.JSON(http.StatusOK, result)
	})

	api.GET("/:hypervisor/listallvms", func(c echo.Context) error {
		name := c.Param("hypervisor")
		authHeader := c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return createErrorResponse(c, http.StatusUnauthorized, err.Error(), "", AuthError)
		}

		if !isAdmin {
			return createErrorResponse(c, http.StatusForbidden, "You do not have permission to access this resource", "", AuthError)
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Unknown hypervisor: "+name, "hypervisor", ValidationError)
		}

		result, err := hv.ListAllVms()
		if err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, err.Error(), "", ServerError)
		}

		var vmMap map[string]interface{}
		if err := json.Unmarshal(result, &vmMap); err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, "Failed to parse JSON: "+err.Error(), "", ServerError)
		}

		return c.JSON(http.StatusOK, vmMap)
	})

	api.GET("/:hypervisor/listvmsuser", func(c echo.Context) error {
		name := c.Param("hypervisor")

		authHeader := c.Request().Header.Get("Authorization")
		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return createErrorResponse(c, http.StatusUnauthorized, err.Error(), "", AuthError)
		}

		db, err := auth.ConnectToDB()
		if err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, "Database connection error: "+err.Error(), "", ServerError)
		}
		defer db.Close()

		rows, err := db.Query("SELECT vmid FROM virtual_machines WHERE user_sid = ?", sid)
		if err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, "DB query error: "+err.Error(), "", ServerError)
		}
		defer rows.Close()

		var vmIDs []int
		for rows.Next() {
			var vmID int
			if err := rows.Scan(&vmID); err != nil {
				return createErrorResponse(c, http.StatusInternalServerError, "DB scan error: "+err.Error(), "", ServerError)
			}
			vmIDs = append(vmIDs, vmID)
		}
		if err := rows.Err(); err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, "DB rows error: "+err.Error(), "", ServerError)
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Unknown hypervisor: "+name, "hypervisor", ValidationError)
		}

		result, err := hv.ListVmsUser(vmIDs, sid)
		if err != nil {
			return createErrorResponse(c, http.StatusInternalServerError, err.Error(), "", ServerError)
		}

		return c.JSON(http.StatusOK, result)
	})

	api.POST("/:hypervisor/create-server", func(c echo.Context) error {
		name := c.Param("hypervisor")

		var requestBody map[string]interface{}
		if err := c.Bind(&requestBody); err != nil {
			return createErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error(), "", ValidationError)
		}

		// Validate VM creation parameters
		if field, message := validateVMCreationParams(requestBody); field != "" {
			return createErrorResponse(c, http.StatusBadRequest, message, field, ValidationError)
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Unknown hypervisor: "+name, "hypervisor", ValidationError)
		}

		authHeader := c.Request().Header.Get("Authorization")
		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return createErrorResponse(c, http.StatusUnauthorized, err.Error(), "", AuthError)
		}

		// Convert validated parameters
		memory := int(requestBody["memory"].(float64))
		cores := int(requestBody["cores"].(float64))
		diskSize := int(requestBody["DiskSize"].(float64))
		vmName := requestBody["name"].(string)
		selectedOs := requestBody["os"].(string)
		subdomain := requestBody["subdomain"].(string)
		serverDescription := requestBody["description"].(string)

		result, err := hv.CreateServer(
			vmName,
			memory,
			cores,
			diskSize,
			html.EscapeString(sid),
			html.EscapeString(givenName),
			html.EscapeString(serverDescription),
			html.EscapeString(selectedOs),
			html.EscapeString(subdomain),
		)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				return createErrorResponse(c, http.StatusConflict, "VM naam of subdomein is al in gebruik", "name", Conflict)
			}
			return createErrorResponse(c, http.StatusInternalServerError, err.Error(), "", ServerError)
		}

		return c.JSON(http.StatusOK, result)
	})

	api.POST("/:hypervisor/delete-server/:vmid", func(c echo.Context) error {
		name := c.Param("hypervisor")
		vmID, err := strconv.Atoi(c.Param("vmid"))
		if err != nil {
			return createErrorResponse(c, http.StatusBadRequest, "Invalid VM ID: "+c.Param("vmid"), "vmid", ValidationError)
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return createErrorResponse(c, http.StatusBadRequest, "Unknown hypervisor: "+name, "hypervisor", ValidationError)
		}

		authHeader := c.Request().Header.Get("Authorization")
		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return createErrorResponse(c, http.StatusUnauthorized, err.Error(), "", AuthError)
		}

		result, err := hv.DeleteServer(vmID, sid)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return createErrorResponse(c, http.StatusNotFound, "VM not found", "vmid", NotFound)
			}
			if strings.Contains(err.Error(), "permission") {
				return createErrorResponse(c, http.StatusForbidden, "You do not have permission to delete this VM", "", AuthError)
			}
			return createErrorResponse(c, http.StatusInternalServerError, err.Error(), "", ServerError)
		}

		return c.JSON(http.StatusOK, result)
	})

	// Create a channel to block the main function
	done := make(chan bool)

	go func() {
		// vmware
		vmware.GenerateVmRoutes()
	}()

	go func() {
		// Start the server
		e.Logger.Fatal(e.Start(":8081"))
		// Signal the channel when the server stops
		done <- true
	}()

	// Block the main function until the channel receives a signal
	<-done
} 