package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"

	hypervisors "CSC-SelfServiceBackend/app"
	"CSC-SelfServiceBackend/app/auth"
	proxmox "CSC-SelfServiceBackend/app/proxmox"
	vmware "CSC-SelfServiceBackend/app/vmwareOld"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

func GetSIDAndName(authorization string) (sid, givenName string, isAdmin bool, err error) {
	if authorization == "" {
		return "", "", false, fmt.Errorf("missing Authorization header")
	}
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authorization, bearerPrefix) {
		return "", "", false, fmt.Errorf("Authorization header missing Bearer prefix")
	}

	tokenString := strings.TrimPrefix(authorization, bearerPrefix)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", "", false, fmt.Errorf("JWT_SECRET environment variable not set")
	}

	t, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !t.Valid {
		return "", "", false, fmt.Errorf("invalid JWT token: %v", err)
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", false, fmt.Errorf("invalid JWT claims")
	}

	sidVal, sidOk := claims["sid"].(string)
	nameVal, nameOk := claims["givenName"].(string)
	isAdmin, isAdminOk := claims["admin"].(bool)
	if !sidOk || !nameOk || !isAdminOk {
		return "", "", false, fmt.Errorf("JWT claims missing sid or givenName or isAdmin")
	}

	return sidVal, nameVal, isAdmin, nil
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
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		fmt.Print(givenName)
		fmt.Print(isAdmin)

		var requestBody map[string]interface{}
		if err := c.Bind(&requestBody); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body: " + err.Error(),
			})
		}

		email, ok := requestBody["email"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid email value"})
		}
		studentID, ok := requestBody["student_id"].(float64)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid student_id value"})
		}
		homeIP, ok := requestBody["home_ip"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid home_ip value"})
		}

		student_id_int := int(studentID)

		result, err := auth.AlterUserInUsersTable(sid, email, student_id_int, homeIP)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, result)
	})

	api.GET("/get-user-info", func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		fmt.Print(givenName)
		fmt.Print(isAdmin)

		if sid == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing sid header"})
		}

		result, err := auth.GetUserDataFromDB(sid)
		if err != nil {
			c.Logger().Errorf("Error getting user data: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, result)
	})

	api.GET("/:hypervisor/listallvms", func(c echo.Context) error {
		name := c.Param("hypervisor")
		authHeader := c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		fmt.Print(givenName)
		fmt.Print(sid)

		if isAdmin == false {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "You do not have permission to access this resource",
			})
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Unknown hypervisor: " + name,
			})
		}

		result, err := hv.ListAllVms()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		// Parse the byte slice into a map
		var vmMap map[string]interface{}
		if err := json.Unmarshal(result, &vmMap); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse JSON: " + err.Error(),
			})
		}

		fmt.Print(vmMap)
		return c.JSON(http.StatusOK, vmMap)
	})

	api.GET("/:hypervisor/listvmsuser", func(c echo.Context) error {
		name := c.Param("hypervisor")

		authHeader := c.Request().Header.Get("Authorization")
		sid, givenName, isAdmin, err := GetSIDAndName(authHeader)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		fmt.Print(givenName)
		fmt.Print(isAdmin)

		db, err := auth.ConnectToDB()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database connection error: " + err.Error()})
		}
		defer db.Close()

		rows, err := db.Query("SELECT vmid FROM virtual_machines WHERE user_sid = ?", sid)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "DB query error: " + err.Error()})
		}
		defer rows.Close()

		var vmIDs []int
		for rows.Next() {
			var vmID int
			if err := rows.Scan(&vmID); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "DB scan error: " + err.Error()})
			}
			vmIDs = append(vmIDs, vmID)
		}
		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "DB rows error: " + err.Error()})
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unknown hypervisor: " + name})
		}

		result, err := hv.ListVmsUser(vmIDs, sid)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, result)
	})

	api.POST("/:hypervisor/create-server", func(c echo.Context) error {
		name := c.Param("hypervisor")

		// Create a map to hold the request body
		var requestBody map[string]interface{}
		if err := c.Bind(&requestBody); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body: " + err.Error(),
			})
		}

		// Log the request body for debugging
		fmt.Printf("Request Body: %+v\n", requestBody)

		hv, ok := hypervisorMap[name]
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Unknown hypervisor: " + name,
			})
		}

		// Convert numeric values to integers
		memory, ok := requestBody["memory"].(float64)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid memory value",
			})
		}

		cores, ok := requestBody["cores"].(float64)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid cores value",
			})
		}
		serverDescription, ok := requestBody["description"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid description value",
			})
		}

		// Change DiskSize type to int64
		fmt.Printf("Raw DiskSize value: %v\n", requestBody["DiskSize"])
		diskSize, ok := requestBody["DiskSize"].(float64)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid DiskSize value: " + strconv.FormatInt(int64(diskSize), 10),
			})
		}

		vmName, ok := requestBody["name"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid name value",
			})
		}

		selectedOs, ok := requestBody["os"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid selected os value",
			})
		}

		subdomain, ok := requestBody["subdomain"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid subdomain value",
			})
		}

		var headers = c.Request().Header
		fmt.Printf("Headers: %+v\n", headers)

		var Authorization = c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(Authorization)
		fmt.Print(isAdmin)

		result, err := hv.CreateServer(vmName, int(memory), int(cores), int(diskSize), html.EscapeString(sid), html.EscapeString(givenName), html.EscapeString(serverDescription), html.EscapeString(selectedOs), html.EscapeString(subdomain))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, result)
	})

	api.POST("/:hypervisor/delete-server/:vmid", func(c echo.Context) error {
		name := c.Param("hypervisor")
		vmID, err := strconv.Atoi(c.Param("vmid"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid VM ID: " + c.Param("vmid"),
			})
		}

		hv, ok := hypervisorMap[name]
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Unknown hypervisor: " + name,
			})
		}

		var Authorization = c.Request().Header.Get("Authorization")

		sid, givenName, isAdmin, err := GetSIDAndName(Authorization)
		fmt.Print(givenName)
		fmt.Print(isAdmin)

		result, err := hv.DeleteServer(vmID, sid)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
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
