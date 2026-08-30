package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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
	tasks    []task.Task
	cursor   int
	path     string
	creating bool
	input    textinput.Model
	showDone bool
}

func NewModel(tasks []task.Task, path string) Model {
	input := textinput.New()
	input.Placeholder = "New todo..."

	return Model{
		tasks:    tasks,
		path:     path,
		input:    input,
		showDone: true,
	}
}

// visible returns the indices into m.tasks that should currently be displayed.
func (m Model) visible() []int {
	indices := make([]int, 0, len(m.tasks))
	for i, t := range m.tasks {
		if m.showDone || !t.Done {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *Model) clampCursor(count int) {
	if m.cursor >= count {
		m.cursor = count - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.creating {
			switch msg.String() {
			case "esc":
				m.creating = false
				m.input.Reset()
				m.input.Blur()

			case "enter":
				if text := strings.TrimSpace(m.input.Value()); text != "" {
					m.tasks = append(m.tasks, task.Create(text))
					if err := task.SaveAll(m.tasks, m.path); err != nil {
						return m, tea.Quit
					}
					m.cursor = len(m.visible()) - 1
				}
				m.creating = false
				m.input.Reset()
				m.input.Blur()

			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

			return m, nil
		}

		visible := m.visible()

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "n":
			m.creating = true
			m.input.Focus()
			return m, textinput.Blink

		case "h":
			m.showDone = !m.showDone
			m.clampCursor(len(m.visible()))

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(visible)-1 {
				m.cursor++
			}

		case "alt+up":
			if m.cursor > 0 {
				a, b := visible[m.cursor], visible[m.cursor-1]
				m.tasks[a], m.tasks[b] = m.tasks[b], m.tasks[a]
				if err := task.SaveAll(m.tasks, m.path); err != nil {
					return m, tea.Quit
				}
				m.cursor--
			}

		case "alt+down":
			if m.cursor < len(visible)-1 {
				a, b := visible[m.cursor], visible[m.cursor+1]
				m.tasks[a], m.tasks[b] = m.tasks[b], m.tasks[a]
				if err := task.SaveAll(m.tasks, m.path); err != nil {
					return m, tea.Quit
				}
				m.cursor++
			}

		case "enter", " ":
			if m.cursor < len(visible) {
				t := &m.tasks[visible[m.cursor]]
				if t.Done {
					t.Done = false
				} else {
					task.MarkDone(t)
				}
				if err := task.SaveAll(m.tasks, m.path); err != nil {
					return m, tea.Quit
				}
				m.clampCursor(len(m.visible()))
			}

		case "d":
			if m.cursor < len(visible) {
				idx := visible[m.cursor]
				m.tasks = append(m.tasks[:idx], m.tasks[idx+1:]...)
				if err := task.SaveAll(m.tasks, m.path); err != nil {
					return m, tea.Quit
				}
				m.clampCursor(len(m.visible()))
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Tasky"))
	b.WriteString("\n")

	visible := m.visible()

	if len(visible) == 0 {
		b.WriteString("No tasks yet.\n")
	}

	for i, idx := range visible {
		t := m.tasks[idx]

		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		text := t.Text
		if t.Done {
			checkbox = "[x]"
			text = doneStyle.Render(t.Text)
		}

		fmt.Fprintf(&b, "%s%s %s\n", cursor, checkbox, text)
	}

	if m.creating {
		fmt.Fprintf(&b, "\n%s\n", m.input.View())
		b.WriteString(helpStyle.Render("enter: save  esc: cancel"))
	} else {
		hideLabel := "hide done"
		if !m.showDone {
			hideLabel = "show done"
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf("up/down: move  alt+up/down: reorder  enter/space: toggle done  n: new  d: delete  h: %s  q: quit", hideLabel)))
	}

	return b.String()
}
