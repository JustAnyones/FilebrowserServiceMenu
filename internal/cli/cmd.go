package cli

import (
	d "filebrowser-service-menu/internal/dbus"
	"fmt"

	"github.com/godbus/dbus/v5"
)

// Sends a notification and prints to console.
func NotifyAndPrint(conn *dbus.Conn, title, message string) {
	fmt.Println(title + ": " + message)
	d.SendNotification(conn, title, message)
}

// Sends an error notification and prints to console.
func NotifyErrorAndPrint(conn *dbus.Conn, err error) {
	NotifyAndPrint(conn, "Error", err.Error())
}
