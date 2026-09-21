// Command demo captures a scripted sequence of the real Tasky model.
//
// It intentionally never starts Bubble Tea's event loop. Sending the same
// tea.KeyMsg values that the TUI receives makes the capture deterministic and
// keeps the demo independent of terminal size or timing.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tasky/internal/task"
	"tasky/internal/tui"
	"tasky/internal/utils"
)

type frame struct {
	Text       string
	DurationMS int
	Caption    string
	Pressed    string
}

func (f frame) MarshalJSON() ([]byte, error) {
	data := map[string]any{
		"text":        f.Text,
		"duration_ms": f.DurationMS,
		"caption":     f.Caption,
	}
	if f.Pressed != "" {
		data["pressed"] = f.Pressed
	}
	return json.Marshal(data)
}

func runeKey(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func specialKey(key tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: key}
}

func altKey(key tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: key, Alt: true}
}

func press(model *tui.Model, msg tea.KeyMsg) {
	_, _ = model.Update(msg)
}

func typeText(model *tui.Model, value string) {
	for _, character := range value {
		press(model, runeKey(string(character)))
	}
}

func clearInput(model *tui.Model, characterCount int) {
	for range characterCount {
		press(model, specialKey(tea.KeyBackspace))
	}
}

func main() {
	outputPath := flag.String("o", "scripts/demo/frames.json", "path for captured frame JSON")
	flag.Parse()

	utils.SetLogEnabled(false)

	tempDir, err := os.MkdirTemp("", "tasky-demo-*")
	if err != nil {
		fatal(err)
	}
	defer os.RemoveAll(tempDir)

	model := tui.NewModel([]task.Group{
		{
			Name: "Launch",
			Tasks: []task.Task{
				{Text: "Ship landing page"},
				{Text: "Reply to beta users"},
				{Text: "Review backlog", Done: true},
			},
		},
		{
			Name:  "Life",
			Tasks: []task.Task{{Text: "Book dentist"}},
		},
		{
			Name:  "Someday",
			Tasks: []task.Task{{Text: "Learn Rust"}},
		},
	}, filepath.Join(tempDir, "tasks.json"))

	frames := make([]frame, 0, 24)
	add := func(durationMS int, caption, pressed string) {
		frames = append(frames, frame{
			Text:       cleanView(model.View()),
			DurationMS: durationMS,
			Caption:    caption,
			Pressed:    pressed,
		})
	}

	add(2200, "Tasks view · start with the active group", "")

	press(&model, specialKey(tea.KeyDown))
	press(&model, specialKey(tea.KeyEnter))
	add(1500, "Complete a task with Enter", "down, enter")

	press(&model, runeKey("h"))
	add(1500, "h hides completed todos", "h")

	press(&model, runeKey("n"))
	add(1100, "n opens the new-task prompt", "n")
	typeText(&model, "Draft launch notes")
	add(1300, "Type directly into the focused prompt", "Draft launch notes")
	press(&model, specialKey(tea.KeyEnter))
	add(1600, "The task is saved in Launch", "enter")

	press(&model, runeKey("a"))
	add(2200, "a shows every enabled group", "a")

	press(&model, runeKey("v"))
	add(1900, "v opens the Groups view", "v")

	press(&model, specialKey(tea.KeyDown))
	press(&model, altKey(tea.KeyDown))
	add(1800, "alt+down reorders the selected group", "down, alt+down")

	press(&model, runeKey("e"))
	add(1000, "e opens group rename", "e")
	clearInput(&model, len("Life"))
	typeText(&model, "Personal")
	press(&model, specialKey(tea.KeyEnter))
	add(1800, "Rename groups without leaving the list", "Personal, enter")

	press(&model, specialKey(tea.KeyUp))
	press(&model, specialKey(tea.KeyEnter))
	add(1900, "Disable a group to hide it from Tasks", "up, enter")

	press(&model, runeKey("v"))
	add(2100, "Tasks omits disabled groups", "v")

	press(&model, runeKey("v"))
	press(&model, runeKey("h"))
	add(2000, "h reveals disabled groups in Groups", "v, h")

	press(&model, specialKey(tea.KeyUp))
	press(&model, specialKey(tea.KeyEnter))
	add(1800, "Enable the selected group again", "up, enter")

	press(&model, runeKey("v"))
	add(2300, "The restored group returns to the all-groups view", "v")

	if err := writeFrames(*outputPath, frames); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %d frames to %s\n", len(frames), *outputPath)
}

func cleanView(value string) string {
	lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	for index := range lines {
		lines[index] = strings.TrimRight(lines[index], " \t")
	}
	return strings.Join(lines, "\n")
}

func writeFrames(path string, frames []frame) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(frames)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
