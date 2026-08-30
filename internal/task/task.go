package task

import (
	"os"

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

func SaveAll(tasks []Task, path string) error {
	if err := utils.WriteJSON(path, tasks); err != nil {
		return err
	}
	utils.Log("Tasks saved successfully")
	return nil
}

func LoadAll(path string) ([]Task, error) {
	var tasks []Task
	if err := utils.ReadJSON(path, &tasks); err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	utils.Log("Tasks loaded successfully")
	return tasks, nil
}
