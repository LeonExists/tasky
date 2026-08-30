package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tasky/internal/task"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

type Model struct {
	tasks  []task.Task
	cursor int
	path   string
}

func NewModel(tasks []task.Task, path string) Model {
	return Model{
		tasks: tasks,
		path:  path,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}

		case "enter", " ":
			if len(m.tasks) > 0 {
				t := &m.tasks[m.cursor]
				if t.Done {
					t.Done = false
				} else {
					task.MarkDone(t)
				}
				if err := task.SaveAll(m.tasks, m.path); err != nil {
					return m, tea.Quit
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Tasky"))
	b.WriteString("\n")

	if len(m.tasks) == 0 {
		b.WriteString("No tasks yet.\n")
	}

	for i, t := range m.tasks {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		title := t.Title
		if t.Done {
			checkbox = "[x]"
			title = doneStyle.Render(t.Title)
		}

		fmt.Fprintf(&b, "%s%s %s\n", cursor, checkbox, title)
	}

	b.WriteString(helpStyle.Render("up/down: move  enter/space: toggle done  q: quit"))

	return b.String()
}
