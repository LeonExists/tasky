package task

import (
	"os"

	"tasky/internal/utils"
)

type Task struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

func Create(text string) Task {
	newTask := Task{
		Text: text,
		Done: false,
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

func MarkDone(t *Task) {
	t.Done = true
	utils.Log("Task marked as done")
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
