package main

import (
	"log"

	"tasky/internal/config"
	"tasky/internal/task"
	"tasky/internal/utils"
)

func main() {
	cfg, err := config.Load("C:/dev/projects/tasky/config.json")
	if err != nil {
		log.Fatal(err)
	}

	tasks, err := task.LoadAll(cfg.TasksPath)
	if err != nil {
		log.Fatal(err)
	}

	t := task.Create("Test", "Description for my test task")
	tasks = append(tasks, t)

	if err := task.SaveAll(tasks, cfg.TasksPath); err != nil {
		log.Fatal(err)
	}

	for _, loaded := range tasks {
		utils.Log("Title: ", loaded.Title)
		utils.Log("Description: ", loaded.Description)
		utils.Log("Done: ", loaded.Done)
	}
}
