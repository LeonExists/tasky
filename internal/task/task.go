package task

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tasky/internal/utils"
)

type Task struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type Group struct {
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

const DefaultGroupName = "General"

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

func SaveGroups(groups []Group, path string) error {
	if err := utils.WriteJSON(path, struct {
		Groups []Group `json:"groups"`
	}{Groups: groups}); err != nil {
		return err
	}
	utils.Log("Task groups saved successfully")
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

func LoadGroups(path string) ([]Group, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultGroups(), nil
		}
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return defaultGroups(), nil
	}

	// Files written by earlier versions contained a flat task array. Keep those
	// files readable and place their tasks in the default group.
	if strings.HasPrefix(strings.TrimSpace(string(data)), "[") {
		var tasks []Task
		if err := json.Unmarshal(data, &tasks); err != nil {
			return nil, err
		}
		return []Group{{Name: DefaultGroupName, Tasks: tasks}}, nil
	}

	var stored struct {
		Groups []Group `json:"groups"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	if len(stored.Groups) == 0 {
		return defaultGroups(), nil
	}

	for i := range stored.Groups {
		stored.Groups[i].Name = strings.TrimSpace(stored.Groups[i].Name)
		if stored.Groups[i].Name == "" {
			stored.Groups[i].Name = fmt.Sprintf("Group %d", i+1)
		}
		if stored.Groups[i].Tasks == nil {
			stored.Groups[i].Tasks = []Task{}
		}
	}

	utils.Log("Task groups loaded successfully")
	return stored.Groups, nil
}

func defaultGroups() []Group {
	return []Group{{Name: DefaultGroupName, Tasks: []Task{}}}
}
