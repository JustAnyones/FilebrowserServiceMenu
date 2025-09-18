package main

import (
	"filebrowser-service-menu/internal/api"
	"filebrowser-service-menu/internal/cli"
	"filebrowser-service-menu/internal/config"
	"filebrowser-service-menu/internal/dbus"
	"flag"
	"fmt"
	"os"
)

func main() {
	filePath := flag.String("filePath", "", "Path to file to upload")
	permanentShare := flag.Bool("permanent", false, "Indicates whether the share has no expiration")
	awoken := flag.Bool("awoken", false, "Indicates if the program was awoken by the service menu")
	flag.Parse()

	expireInDays := 30
	if *permanentShare {
		expireInDays = 0
	}

	// Load config from file
	fileCfg, err := config.NewConfigFromFile()
	if err != nil {
		fmt.Println("Error loading config from file:", err)
		return
	}

	// Connect to session bus
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Println("Failed to connect to session bus:", err)
		return
	}
	defer conn.Close()

	if *awoken {
		cli.NotifyAndPrint(conn, "Info", "Not yet implemented.")
		return
	}

	// Check if file path is provided
	if *filePath == "" {
		cli.NotifyErrorAndPrint(conn, fmt.Errorf("no file specified. Please specify a file to upload using --filePath"))
		return
	}

	// Check if file exists
	_, err = os.Stat(*filePath)
	if os.IsNotExist(err) {
		cli.NotifyErrorAndPrint(conn, fmt.Errorf("file does not exist: %s", *filePath))
		return
	} else if err != nil {
		cli.NotifyErrorAndPrint(conn, fmt.Errorf("error checking file: %v", err))
		return
	}

	// Log in
	fmt.Println("Logging in")
	session, err := api.Login(fileCfg.InstanceUrl, fileCfg.Username, fileCfg.Password)
	if err != nil {
		cli.NotifyErrorAndPrint(conn, fmt.Errorf("failed to login: %v", err))
		return
	}

	// Perform the actual upload
	cli.NotifyAndPrint(conn, "Uploading", "Uploading "+*filePath)
	link, err := session.Upload(*filePath, expireInDays)
	if err != nil {
		cli.NotifyErrorAndPrint(conn, fmt.Errorf("failed to upload: %v", err))
		return
	}
	dbus.SendLinkNotification(conn, "Upload complete", "File uploaded successfully\n"+link, link)
}
