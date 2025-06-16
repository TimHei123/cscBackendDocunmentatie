package vmware

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetNotifications retrieves all notifications for the current user
func GetNotifications(c echo.Context) error {
	studentId, _, _, _ := getUserAssociatedWithJWT(c)
	db, err := connectToDB()
	if err != nil {
		log.Println("Error connecting to database: ", err)
		return c.String(http.StatusInternalServerError, "Internal server error")
	}

	notifications := getNotificationsAssociatedWithUser(db, studentId)

	return c.JSON(http.StatusOK, notifications)
}

// ChangeReadStatusOfNotification toggles the read status of a notification
func ChangeReadStatusOfNotification(c echo.Context) error {
	studentId, _, _, _ := getUserAssociatedWithJWT(c)
	db, err := connectToDB()
	if err != nil {
		log.Println("Error connecting to database: ", err)
		return c.String(http.StatusInternalServerError, "Internal server error")
	}

	notificationId := c.Param("id")

	_, err = db.Exec("UPDATE notifications SET read_notif = IF(read_notif = 1, 0, 1) WHERE user_id = ? AND id = ?", studentId, notificationId)
	if err != nil {
		log.Println("Error executing query: ", err)
		return c.String(http.StatusInternalServerError, "Internal server error")
	}

	return c.String(http.StatusOK, "Notification status updated")
}
