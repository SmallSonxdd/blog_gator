package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbURL    string `json:"db_url"`
	Username string `json:"current_user_name"`
}

func Read() (Config, error) {
	full_path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(full_path)
	if err != nil {
		return Config{}, err
	}

	var rtr_config Config
	if err := json.Unmarshal(data, &rtr_config); err != nil {
		return Config{}, err
	}

	return rtr_config, nil
}

func (con *Config) SetUser(username string) error {
	fmt.Printf("Setting user to: %s\n", username)
	con.Username = username
	return write(*con)
}

func getConfigFilePath() (string, error) {
	dir_str, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	full_path := filepath.Join(dir_str, configFileName)
	return full_path, nil
}

func write(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	full_path, err := getConfigFilePath()
	if err != nil {
		return err
	}
	err = os.WriteFile(full_path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
