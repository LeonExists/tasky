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
	titleStyle         = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	navigationStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedGroupStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	groupHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	cursorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	doneStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)
	emptyTaskStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

type taskRef struct {
	groupIndex int
	taskIndex  int
}

type Model struct {
	groups        []task.Group
	activeGroup   int
	cursor        int
	path          string
	creating      bool
	creatingGroup bool
	editing       bool
	editGroup     int
	editIndex     int
	input         textinput.Model
	showDone      bool
	allGroups     bool
}

func NewModel(groups []task.Group, path string) Model {
	if len(groups) == 0 {
		groups = []task.Group{{Name: task.DefaultGroupName, Tasks: []task.Task{}}}
	}

	modelGroups := make([]task.Group, len(groups))
	copy(modelGroups, groups)
	for i := range modelGroups {
		modelGroups[i].Name = strings.TrimSpace(modelGroups[i].Name)
		if modelGroups[i].Name == "" {
			modelGroups[i].Name = fmt.Sprintf("Group %d", i+1)
		}
		if modelGroups[i].Tasks == nil {
			modelGroups[i].Tasks = []task.Task{}
		}
	}

	input := textinput.New()
	input.Placeholder = "New todo..."

	return Model{
		groups:   modelGroups,
		path:     path,
		input:    input,
		showDone: true,
	}
}

func (m Model) visibleTasksForGroup(groupIndex int) []taskRef {
	if groupIndex < 0 || groupIndex >= len(m.groups) {
		return nil
	}

	indices := make([]taskRef, 0, len(m.groups[groupIndex].Tasks))
	for taskIndex, currentTask := range m.groups[groupIndex].Tasks {
		if m.showDone || !currentTask.Done {
			indices = append(indices, taskRef{groupIndex: groupIndex, taskIndex: taskIndex})
		}
	}
	return indices
}

func (m Model) visibleTasks() []taskRef {
	if !m.allGroups {
		return m.visibleTasksForGroup(m.activeGroup)
	}

	indices := make([]taskRef, 0)
	for groupIndex := range m.groups {
		indices = append(indices, m.visibleTasksForGroup(groupIndex)...)
	}
	return indices
}

func (m Model) taskPosition(ref taskRef) int {
	for position, visibleRef := range m.visibleTasks() {
		if visibleRef == ref {
			return position
		}
	}
	return -1
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.creating || m.creatingGroup || m.editing {
			return m.updateInput(msg)
		}

		visible := m.visibleTasks()

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "n":
			m.creating = true
			m.input.Reset()
			m.input.Placeholder = "New todo..."
			m.input.Focus()
			return m, textinput.Blink

		case "c", "g":
			m.creatingGroup = true
			m.input.Reset()
			m.input.Placeholder = "Group name..."
			m.input.Focus()
			return m, textinput.Blink

		case "a", "0":
			m.allGroups = !m.allGroups
			m.clampCursor(len(m.visibleTasks()))

		case "left", "[", "shift+tab":
			m.switchGroup(-1)

		case "right", "]", "tab":
			m.switchGroup(1)

		case "e":
			if m.cursor < len(visible) {
				ref := visible[m.cursor]
				m.editing = true
				m.editGroup = ref.groupIndex
				m.editIndex = ref.taskIndex
				m.input.SetValue(m.groups[ref.groupIndex].Tasks[ref.taskIndex].Text)
				m.input.CursorEnd()
				m.input.Focus()
				return m, textinput.Blink
			}

		case "h":
			m.showDone = !m.showDone
			m.clampCursor(len(m.visibleTasks()))

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
				current, previous := visible[m.cursor], visible[m.cursor-1]
				if current.groupIndex == previous.groupIndex {
					m.groups[current.groupIndex].Tasks[current.taskIndex], m.groups[previous.groupIndex].Tasks[previous.taskIndex] =
						m.groups[previous.groupIndex].Tasks[previous.taskIndex], m.groups[current.groupIndex].Tasks[current.taskIndex]
					if err := m.save(); err != nil {
						return m, tea.Quit
					}
					m.cursor--
				}
			}

		case "alt+down":
			if m.cursor < len(visible)-1 {
				current, next := visible[m.cursor], visible[m.cursor+1]
				if current.groupIndex == next.groupIndex {
					m.groups[current.groupIndex].Tasks[current.taskIndex], m.groups[next.groupIndex].Tasks[next.taskIndex] =
						m.groups[next.groupIndex].Tasks[next.taskIndex], m.groups[current.groupIndex].Tasks[current.taskIndex]
					if err := m.save(); err != nil {
						return m, tea.Quit
					}
					m.cursor++
				}
			}

		case "enter", " ":
			if m.cursor < len(visible) {
				ref := visible[m.cursor]
				currentTask := &m.groups[ref.groupIndex].Tasks[ref.taskIndex]
				if currentTask.Done {
					currentTask.Done = false
				} else {
					task.MarkDone(currentTask)
				}
				if err := m.save(); err != nil {
					return m, tea.Quit
				}
				m.clampCursor(len(m.visibleTasks()))
			}

		case "d":
			if m.cursor < len(visible) {
				ref := visible[m.cursor]
				groupTasks := m.groups[ref.groupIndex].Tasks
				m.groups[ref.groupIndex].Tasks = append(groupTasks[:ref.taskIndex], groupTasks[ref.taskIndex+1:]...)
				if err := m.save(); err != nil {
					return m, tea.Quit
				}
				m.clampCursor(len(m.visibleTasks()))
			}
		}
	}

	return m, nil
}

func (m *Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.cancelInput()

	case "enter":
		value := strings.TrimSpace(m.input.Value())
		changed := false

		switch {
		case m.creatingGroup:
			if value != "" {
				m.groups = append(m.groups, task.Group{Name: value, Tasks: []task.Task{}})
				m.activeGroup = len(m.groups) - 1
				m.allGroups = false
				m.cursor = 0
				changed = true
			}

		case m.editing:
			if value != "" {
				m.groups[m.editGroup].Tasks[m.editIndex].Text = value
				changed = true
			}

		case m.creating:
			if value != "" && len(m.groups) > 0 {
				groupIndex := m.activeGroup
				if m.allGroups {
					visible := m.visibleTasks()
					if m.cursor < len(visible) {
						groupIndex = visible[m.cursor].groupIndex
					}
				}
				m.groups[groupIndex].Tasks = append(m.groups[groupIndex].Tasks, task.Create(value))
				m.activeGroup = groupIndex
				m.cursor = m.taskPosition(taskRef{
					groupIndex: groupIndex,
					taskIndex:  len(m.groups[groupIndex].Tasks) - 1,
				})
				changed = true
			}
		}

		if changed {
			if err := m.save(); err != nil {
				return m, tea.Quit
			}
		}
		m.cancelInput()

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) cancelInput() {
	m.creating = false
	m.creatingGroup = false
	m.editing = false
	m.input.Reset()
	m.input.Placeholder = "New todo..."
	m.input.Blur()
}

func (m *Model) switchGroup(direction int) {
	if len(m.groups) == 0 {
		return
	}

	m.activeGroup = (m.activeGroup + direction) % len(m.groups)
	if m.activeGroup < 0 {
		m.activeGroup += len(m.groups)
	}
	m.allGroups = false
	m.clampCursor(len(m.visibleTasks()))
}

func (m Model) save() error {
	return task.SaveGroups(m.groups, m.path)
}

func (m Model) groupNavigation() string {
	items := make([]string, 0, len(m.groups)+1)
	if m.allGroups {
		items = append(items, selectedGroupStyle.Render("[All]"))
	} else {
		items = append(items, navigationStyle.Render("All"))
	}

	for groupIndex, group := range m.groups {
		if !m.allGroups && groupIndex == m.activeGroup {
			items = append(items, selectedGroupStyle.Render("["+group.Name+"]"))
		} else {
			items = append(items, navigationStyle.Render(group.Name))
		}
	}
	return strings.Join(items, "  ")
}

func (m Model) writeTask(b *strings.Builder, ref taskRef, position int) {
	currentTask := m.groups[ref.groupIndex].Tasks[ref.taskIndex]

	cursor := "  "
	if position == m.cursor {
		cursor = cursorStyle.Render("> ")
	}

	checkbox := "[ ]"
	text := currentTask.Text
	if currentTask.Done {
		checkbox = "[x]"
		text = doneStyle.Render(currentTask.Text)
	}

	fmt.Fprintf(b, "%s%s %s\n", cursor, checkbox, text)
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Tasky"))
	b.WriteString("\n")
	b.WriteString(m.groupNavigation())
	b.WriteString("\n\n")

	visible := m.visibleTasks()
	if m.allGroups {
		position := 0
		for groupIndex, group := range m.groups {
			b.WriteString(groupHeaderStyle.Render(group.Name))
			b.WriteString("\n")

			groupTasks := m.visibleTasksForGroup(groupIndex)
			if len(groupTasks) == 0 {
				b.WriteString(emptyTaskStyle.Render("  No tasks yet."))
				b.WriteString("\n")
			} else {
				for _, ref := range groupTasks {
					m.writeTask(&b, ref, position)
					position++
				}
			}

			if groupIndex < len(m.groups)-1 {
				b.WriteString("\n")
			}
		}
	} else {
		groupName := task.DefaultGroupName
		if m.activeGroup >= 0 && m.activeGroup < len(m.groups) {
			groupName = m.groups[m.activeGroup].Name
		}
		b.WriteString(groupHeaderStyle.Render(groupName))
		b.WriteString("\n")

		if len(visible) == 0 {
			b.WriteString(emptyTaskStyle.Render("No tasks yet."))
			b.WriteString("\n")
		} else {
			for position, ref := range visible {
				m.writeTask(&b, ref, position)
			}
		}
	}

	if m.creatingGroup {
		fmt.Fprintf(&b, "\n%s\n", m.input.View())
		b.WriteString(helpStyle.Render("enter: create group  esc: cancel"))
	} else if m.creating || m.editing {
		fmt.Fprintf(&b, "\n%s\n", m.input.View())
		b.WriteString(helpStyle.Render("enter: save  esc: cancel"))
	} else {
		hideLabel := "hide done"
		if !m.showDone {
			hideLabel = "show done"
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf(
			"left/right/tab: switch group  a: all groups  c/g: new group  up/down: move  alt+up/down: reorder  enter/space: toggle done  n: new  e: edit  d: delete  h: %s  q: quit",
			hideLabel,
		)))
	}

	return b.String()
}
