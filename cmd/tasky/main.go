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
	cfgPath, err := config.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	utils.SetLogEnabled(cfg.LogEnabled)

	groups, err := task.LoadGroups(cfg.TasksPath)
	if err != nil {
		log.Fatal(err)
	}

	m := tui.NewModel(groups, cfg.TasksPath)
	if _, err := tea.NewProgram(&m).Run(); err != nil {
		fmt.Println("Error running Tasky:", err)
		os.Exit(1)
	}
}
