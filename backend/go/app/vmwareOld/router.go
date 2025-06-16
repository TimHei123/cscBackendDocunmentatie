package vmware

import (
	"net/http"

	"github.com/labstack/echo/v4/middleware"

	"github.com/labstack/echo/v4"
)

func GenerateVmRoutes() {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowHeaders: []string{"*"},
	}))
	e.GET("/", func(c echo.Context) error { return c.String(http.StatusTeapot, "I'm a teapot") })

	s := e.Group("/servers")
	s.Use(checkIfLoggedIn)

	s.GET("", GetServers)
	s.POST("", CreateServer)

	// s.PATCH("/:id", UpdateServer)

	s.POST("/power/:id/:status", PowerServer)

	s.GET("/:id", GetServers)

	s.DELETE("/:id", DeleteServer)

	d := e.Group("/dns")
	d.Use(checkIfLoggedIn)

	d.GET("", GetDnsZones)
	d.GET("/:serverId", GetDnsRecordsForServer)
	d.POST("/:serverId", CreateDnsRecord)
	d.PATCH("/:recordId", UpdateDnsRecord)
	d.DELETE("/:serverId", DeleteDnsRecord)

	// TODO: array met template IDs cachen (fetchTemplateLibraryIdsFromVCenter)
	// TODO: JSON het zelfde maken als de Laravel JSON
	/*
	   {
	       "UBUNTU TEMPLATE": {
	           "storage": 20,
	           "memory": 1,
	           "os": "UBUNTU_64"
	       },
	       "OICT-AUTO-Template": {
	           "storage": 20,
	           "memory": 1,
	           "os": "UBUNTU_64"
	       },
	       "OICT-AUTO-DEBIAN": {
	           "storage": 20,
	           "memory": 1,
	           "os": "UBUNTU_64"
	       }
	   }
	*/
	e.GET("/templates", GetTemplates)

	g := e.Group("/admin")
	g.Use(checkIfLoggedInAsAdmin)

	g.POST("/ipAddresses", CreateIpAdress)

	// force the templates to be re-cached
	g.GET("/templates/refresh", RefreshTemplates)
	g.GET("/dataStores/refresh", RefreshDataStores)

	a := e.Group("/auth")

	a.POST("/login", Login)
	a.POST("/resetRequest", ResetRequest)
	a.POST("/resetPassword", ResetPassword)

	e.GET("checkIfLoginTokenIsValid", CheckIfLoginTokenIsValid)

	n := e.Group("/notifications")
	n.Use(checkIfLoggedIn)

	n.GET("", GetNotifications)
	n.PATCH("/:id", ChangeReadStatusOfNotification)

	tickets := e.Group("/tickets")
	tickets.Use(checkIfLoggedIn)

	tickets.GET("", GetTickets)
	tickets.POST("", CreateTicket)
	tickets.GET("/:id", GetTickets)
	tickets.PATCH("/:id", UpdateTicket)
	tickets.DELETE("/:id", DeleteTicket)

	e.Start(":8080")
}
