package utils

import "fmt"

// === Uitls Settings ===
var logEnabled bool = true

func Log(args ...any) {
	if !logEnabled {
		return
	}

	message := fmt.Sprint(args...)
	fmt.Println("Tasky:", message)
}
