package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

// All configuration options for the application.
// All fields must be set in the config file.
type Config struct {
	InstanceUrl string
	Username    string
	Password    string
}

// NewConfigFromFile loads the configuration from a file located at
// UserConfigDir()/filebrowser-service-menu/config.toml
//
// If the file does not exist or cannot be read, it returns an error.
func NewConfigFromFile() (*Config, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, errors.New("Could not get user config dir: " + err.Error())
	}

	path := filepath.Join(dir, "justanyone", "filebrowser-service-menu", "config.toml")
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return nil, errors.New("Config file does not exist at " + path)
	} else if err != nil {
		return nil, errors.New("Could not stat config file: " + err.Error())
	}

	// Unmarshal the file
	var cfg Config
	_, err = toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, errors.New("Could not decode config file: " + err.Error())
	}

	// Ensure that all fields are set in the config
	typ, val := reflect.TypeOf(cfg), reflect.ValueOf(cfg)
	for i := range typ.NumField() {
		// This check only runs for string fields
		if typ.Field(i).Type.Kind() != reflect.String {
			return nil, fmt.Errorf("Config field %s is not a string", typ.Field(i).Name)
		}
		// Cast to string
		valueAsString, err := val.Field(i).Interface().(string)
		if !err {
			return nil, fmt.Errorf("Config field %s is not a string", typ.Field(i).Name)
		}
		// Check if empty or only whitespace
		if strings.TrimSpace(valueAsString) == "" {
			return nil, fmt.Errorf("Config field %s is not set", typ.Field(i).Name)
		}
	}

	return &cfg, nil
}
