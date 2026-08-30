package config

import (
	"os"
	"path/filepath"

	"tasky/internal/utils"
)

type Config struct {
	TasksPath  string `json:"tasksPath"`
	LogEnabled bool   `json:"logEnabled"`
}

// DefaultPath returns the config file location used when none is specified:
// a config.json inside a .tasky directory in the user's home directory.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tasky", "config.json"), nil
}

func Load(path string) (Config, error) {
	var cfg Config
	if err := utils.ReadJSON(path, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.TasksPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.TasksPath = filepath.Join(home, ".tasky", "tasks.json")
		}
	}

	utils.Log("Config loaded successfully")
	return cfg, nil
}
