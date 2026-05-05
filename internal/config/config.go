package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

// Config holds persistent application settings stored in the user's home directory.
type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

// ReadConfig loads configuration from the ~/.gatorconfig.json file.
func ReadConfig() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// SetUser updates the current user and persists the config file.
func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	return writeConfig(*c)
}

// SetDBURL updates the database connection URL and persists the config file.
func (c *Config) SetDBURL(url string) error {
	c.DBURL = url
	return writeConfig(*c)
}

// GetUser returns the configured current user name.
func (c *Config) GetUser() string {
	return c.CurrentUserName
}

// GetDBURL returns the configured database connection URL.
func (c *Config) GetDBURL() string {
	return c.DBURL
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configFileName), nil
}

func writeConfig(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
