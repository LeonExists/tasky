package main

import "tasky/internal/task"

func main() {
	t := task.Create("Test", "Description for my test task")
	task.Load(t)
}
