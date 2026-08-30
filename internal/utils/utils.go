package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// === Uitls Settings ===
var logEnabled bool = true

func SetLogEnabled(enabled bool) {
	logEnabled = enabled
}

// === Logging ===
func Log(args ...any) {
	if !logEnabled {
		return
	}

	message := fmt.Sprint(args...)
	fmt.Println("Tasky:", message)
}

// === Write & Read JSON ===
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, v)
}
