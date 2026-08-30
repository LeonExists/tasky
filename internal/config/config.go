package config

import "tasky/internal/utils"

type Config struct {
	TasksPath string `json:"tasksPath"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if err := utils.ReadJSON(path, &cfg); err != nil {
		return Config{}, err
	}
	utils.Log("Config loaded successfully")
	return cfg, nil
}
