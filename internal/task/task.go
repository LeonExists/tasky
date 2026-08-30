package task

import (
	"tasky/internal/utils"
)

type Task struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

func Create(title string, description string) Task {
	newTask := Task{
		Title:       title,
		Description: description,
		Done:        false,
	}

	utils.Log("Task created successfully")

	return newTask
}

func Load(task Task) {
	utils.Log("Successfully loaded task...")
	utils.Log("Title: ", task.Title)
	utils.Log("Description: ", task.Description)
	utils.Log("Done: ", task.Done)
}
