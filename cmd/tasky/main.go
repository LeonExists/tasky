package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tasky/internal/config"
	"tasky/internal/task"
	"tasky/internal/tui"
	"tasky/internal/utils"
)

func main() {
	cfg, err := config.Load("C:/dev/projects/tasky/config.json")
	if err != nil {
		log.Fatal(err)
	}
	utils.SetLogEnabled(cfg.LogEnabled)

	tasks, err := task.LoadAll(cfg.TasksPath)
	if err != nil {
		log.Fatal(err)
	}

	m := tui.NewModel(tasks, cfg.TasksPath)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running Tasky:", err)
		os.Exit(1)
	}
}
