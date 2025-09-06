package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const NAME = "lt.svetikas.FilebrowserServiceMenu"

func copyToClipboard(conn *dbus.Conn, text string) error {
	obj := conn.Object("org.kde.klipper", "/klipper")
	call := obj.Call("org.kde.klipper.klipper.setClipboardContents", 0, text)
	return call.Err
}

func SendNotification(conn *dbus.Conn, title, message string) {
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		"Share via Filebrowser",   // App name
		uint32(0),                 // Notification ID (0 means new notification)
		"",                        // Icon (empty string means no icon)
		title,                     // Summary (title)
		message,                   // Body (message)
		[]string{},                // Actions (optional)
		map[string]dbus.Variant{}, // Hints (optional)
		int32(7000),               // Timeout (in milliseconds)
	)
	fmt.Println("Sending notification:", title, message)
	if call.Err != nil {
		fmt.Println("Failed to send notification:", call.Err)
		return
	}
}

func SendLinkNotification(conn *dbus.Conn, title, message, link string) {
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	actions := []string{"copy", "Copy Link"}
	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		"Share via Filebrowser",
		uint32(0),
		"",
		title,
		message,
		actions,
		map[string]dbus.Variant{
			"desktop-entry": dbus.MakeVariant(NAME), // App identifier
		},
		int32(5000),
	)
	if call.Err != nil {
		fmt.Println("Failed to send notification:", call.Err)
		return
	}

	var notificationID uint32
	err := call.Store(&notificationID)
	if err != nil {
		fmt.Println("Failed to get notification ID:", err)
		return
	}

	fmt.Println("Notification sent. Waiting for action...")

	// Listen for action responses
	signal := make(chan *dbus.Signal, 1)
	conn.Signal(signal)

	// ActionInvoked signal
	conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/Notifications"),
		dbus.WithMatchInterface("org.freedesktop.Notifications"),
		dbus.WithMatchMember("ActionInvoked"),
	)
	// NotificationClosed signal
	conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/Notifications"),
		dbus.WithMatchInterface("org.freedesktop.Notifications"),
		dbus.WithMatchMember("NotificationClosed"),
	)

	// TODO: currently waiting forever, can't be bothered
	// to figure out why actions get removed when process dies on KDE
	for {
		select {
		case sig := <-signal:
			// If notification was closed
			if sig.Name == "org.freedesktop.Notifications.NotificationClosed" {
				fmt.Println("Notification closed.")
				return
			}

			if len(sig.Body) < 2 {
				continue
			}
			id, ok := sig.Body[0].(uint32)
			action, actionOk := sig.Body[1].(string)
			if !ok || !actionOk {
				continue
			}

			// If "copy" button was clicked
			if id == notificationID && action == "copy" {
				copyToClipboard(conn, link)
				return
			}
			//case <-time.After(time.Duration(timeout) * 2 * time.Second):
			//	fmt.Println("No action taken.")
			//	return
			//}
		}
	}

	/*err = copyToClipboard(conn, link)
	if err != nil {
		fmt.Println("Failed to copy to clipboard:", err)
		return
	}*/
}

func SessionBus() (*dbus.Conn, error) {
	return dbus.SessionBus()
}
