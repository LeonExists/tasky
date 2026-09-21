package tui

import (
	"fmt"
	"strconv"
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
	groupMetaStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	groupEnabledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	cursorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	doneStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)
	emptyTaskStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	disabledStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type taskRef struct {
	groupIndex int
	taskIndex  int
}

type viewMode string

const (
	tasksView  viewMode = "tasks"
	groupsView viewMode = "groups"
)

type Model struct {
	groups         []task.Group
	activeGroup    int // selected group in Tasks view; always enabled or -1
	groupSelection int // selected group in Groups view; underlying slice index
	groupCursor    int // position of groupSelection in the filtered Groups list
	cursor         int
	path           string
	creating       bool
	creatingGroup  bool
	editing        bool
	editingGroup   bool
	editGroup      int
	editIndex      int
	input          textinput.Model
	showDone       bool
	showDisabled   bool
	allGroups      bool
	view           viewMode
	confirmDelete  bool
	deleteGroup    int
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

	model := Model{
		groups:         modelGroups,
		activeGroup:    -1,
		groupSelection: -1,
		path:           path,
		input:          input,
		showDone:       true,
		view:           tasksView,
	}
	model.ensureTaskGroup()
	model.groupSelection = model.activeGroup
	model.view = groupsView
	model.normalizeGroupSelection()
	model.view = tasksView
	return model
}

func (m Model) enabledGroupIndices() []int {
	indices := make([]int, 0, len(m.groups))
	for groupIndex, group := range m.groups {
		if !group.Disabled {
			indices = append(indices, groupIndex)
		}
	}
	return indices
}

// groupIndicesForCurrentView returns the group rows that can be selected in
// the current view. Tasks always filters disabled groups. Groups can include
// them when showDisabled is enabled.
func (m Model) groupIndicesForCurrentView() []int {
	if m.view == groupsView && m.showDisabled {
		indices := make([]int, len(m.groups))
		for groupIndex := range m.groups {
			indices[groupIndex] = groupIndex
		}
		return indices
	}
	return m.enabledGroupIndices()
}

func (m Model) groupPosition(groupIndex int) int {
	for position, index := range m.groupIndicesForCurrentView() {
		if index == groupIndex {
			return position
		}
	}
	return -1
}

// normalizeGroupSelection keeps groupSelection as an underlying slice index,
// while making sure it refers to a selectable group in the Groups view.
// Keeping the underlying index means reordering and filtering cannot make the
// selected group silently jump to a different group.
func (m *Model) normalizeGroupSelection() {
	indices := m.groupIndicesForCurrentView()
	if len(indices) == 0 {
		m.groupCursor = 0
		m.clampCursor(len(m.visibleTasks()))
		return
	}

	if m.groupPosition(m.groupSelection) < 0 {
		previous := m.groupSelection
		m.groupSelection = indices[0]
		if previous >= 0 {
			for _, index := range indices {
				if index >= previous {
					m.groupSelection = index
					break
				}
			}
		}
	}

	m.groupCursor = m.groupPosition(m.groupSelection)
	if m.groupCursor < 0 {
		m.groupCursor = 0
	}
	m.clampCursor(len(m.visibleTasks()))
}

func (m Model) visibleTasksForGroup(groupIndex int) []taskRef {
	if groupIndex < 0 || groupIndex >= len(m.groups) || m.groups[groupIndex].Disabled {
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
	for _, groupIndex := range m.enabledGroupIndices() {
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
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.creating || m.creatingGroup || m.editing || m.editingGroup {
			return m.updateInput(msg)
		}

		if m.confirmDelete {
			return m.updateDeleteConfirmation(msg)
		}

		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "v" {
			m.toggleView()
			return m, nil
		}

		if m.view == groupsView {
			return m.updateGroupsView(msg)
		}
		return m.updateTasksView(msg)
	}

	return m, nil
}

func (m *Model) updateTasksView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleTasks()

	switch msg.String() {
	case "n":
		if len(m.enabledGroupIndices()) > 0 {
			m.ensureTaskGroup()
			m.creating = true
			m.input.Reset()
			m.input.Placeholder = "New todo..."
			m.input.Focus()
			return m, textinput.Blink
		}

	case "c", "g":
		m.startGroupCreation()
		return m, textinput.Blink

	case "a", "0":
		m.allGroups = !m.allGroups
		m.normalizeGroupSelection()
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

	return m, nil
}

func (m *Model) updateGroupsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n", "c", "g":
		m.startGroupCreation()
		return m, textinput.Blink

	case "up", "k":
		m.moveGroupSelection(-1)

	case "down", "j":
		m.moveGroupSelection(1)

	case "e":
		if groupIndex := m.selectedGroupIndex(); groupIndex >= 0 {
			m.editingGroup = true
			m.editGroup = groupIndex
			m.input.SetValue(m.groups[groupIndex].Name)
			m.input.CursorEnd()
			m.input.Placeholder = "Group name..."
			m.input.Focus()
			return m, textinput.Blink
		}

	case "d":
		if groupIndex := m.selectedGroupIndex(); groupIndex >= 0 {
			m.confirmDelete = true
			m.deleteGroup = groupIndex
		}

	case "alt+up":
		if err := m.reorderGroup(-1); err != nil {
			return m, tea.Quit
		}

	case "alt+down":
		if err := m.reorderGroup(1); err != nil {
			return m, tea.Quit
		}

	case "enter", " ":
		if groupIndex := m.selectedGroupIndex(); groupIndex >= 0 {
			m.groups[groupIndex].Disabled = !m.groups[groupIndex].Disabled
			if m.groups[groupIndex].Disabled && m.activeGroup == groupIndex {
				m.selectEnabledGroup(groupIndex)
			} else if !m.groups[groupIndex].Disabled && m.activeGroup < 0 {
				m.activeGroup = groupIndex
			}
			m.normalizeGroupSelection()
			if err := m.save(); err != nil {
				return m, tea.Quit
			}
		}

	case "h":
		m.showDisabled = !m.showDisabled
		m.normalizeGroupSelection()
	}

	return m, nil
}

func (m *Model) updateDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.confirmDelete = false
		if err := m.deleteGroupAt(m.deleteGroup); err != nil {
			return m, tea.Quit
		}
	case "n", "esc":
		m.confirmDelete = false
		m.deleteGroup = -1
	}
	return m, nil
}

func (m *Model) startGroupCreation() {
	m.creatingGroup = true
	m.input.Reset()
	m.input.Placeholder = "Group name..."
	m.input.Focus()
}

func (m *Model) toggleView() {
	if m.view == groupsView {
		m.view = tasksView
		m.ensureTaskGroup()
		m.clampCursor(len(m.visibleTasks()))
		return
	}

	m.view = groupsView
	m.normalizeGroupSelection()
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
				newGroup := len(m.groups) - 1
				m.groupSelection = newGroup
				m.groupCursor = 0
				if m.view == tasksView {
					m.activeGroup = newGroup
					m.allGroups = false
					m.cursor = 0
				} else if m.activeGroup < 0 {
					m.activeGroup = newGroup
				}
				m.normalizeGroupSelection()
				changed = true
			}

		case m.editingGroup:
			if value != "" && m.editGroup >= 0 && m.editGroup < len(m.groups) {
				m.groups[m.editGroup].Name = value
				m.groupSelection = m.editGroup
				m.normalizeGroupSelection()
				changed = true
			}

		case m.editing:
			if value != "" && m.editGroup >= 0 && m.editGroup < len(m.groups) &&
				m.editIndex >= 0 && m.editIndex < len(m.groups[m.editGroup].Tasks) {
				m.groups[m.editGroup].Tasks[m.editIndex].Text = value
				changed = true
			}

		case m.creating:
			if value != "" {
				groupIndex := m.taskCreationGroup()
				if groupIndex >= 0 {
					m.groups[groupIndex].Tasks = append(m.groups[groupIndex].Tasks, task.Create(value))
					m.activeGroup = groupIndex
					m.cursor = m.taskPosition(taskRef{
						groupIndex: groupIndex,
						taskIndex:  len(m.groups[groupIndex].Tasks) - 1,
					})
					changed = true
				}
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
	m.editingGroup = false
	m.editGroup = -1
	m.editIndex = -1
	m.input.Reset()
	m.input.Placeholder = "New todo..."
	m.input.Blur()
}

func (m *Model) taskCreationGroup() int {
	m.ensureTaskGroup()
	groupIndex := m.activeGroup
	if m.allGroups {
		visible := m.visibleTasks()
		if m.cursor < len(visible) {
			groupIndex = visible[m.cursor].groupIndex
		}
	}
	if groupIndex < 0 || groupIndex >= len(m.groups) || m.groups[groupIndex].Disabled {
		return -1
	}
	return groupIndex
}

func (m *Model) ensureTaskGroup() {
	indices := m.enabledGroupIndices()
	if len(indices) == 0 {
		m.activeGroup = -1
		m.cursor = 0
		return
	}
	if m.activeGroup < 0 || m.activeGroup >= len(m.groups) || m.groups[m.activeGroup].Disabled {
		m.activeGroup = indices[0]
	}
}

func (m *Model) selectEnabledGroup(preferred int) {
	indices := m.enabledGroupIndices()
	if len(indices) == 0 {
		m.activeGroup = -1
		return
	}

	for _, index := range indices {
		if index >= preferred {
			m.activeGroup = index
			return
		}
	}
	m.activeGroup = indices[len(indices)-1]
}

func (m *Model) switchGroup(direction int) {
	indices := m.enabledGroupIndices()
	if len(indices) == 0 {
		m.activeGroup = -1
		m.allGroups = false
		m.cursor = 0
		return
	}

	position := -1
	for candidatePosition, index := range indices {
		if index == m.activeGroup {
			position = candidatePosition
			break
		}
	}
	if position < 0 {
		position = 0
	}

	position = (position + direction) % len(indices)
	if position < 0 {
		position += len(indices)
	}
	m.activeGroup = indices[position]
	m.allGroups = false
	m.clampCursor(len(m.visibleTasks()))
}

func (m *Model) moveGroupSelection(direction int) {
	indices := m.groupIndicesForCurrentView()
	if len(indices) == 0 {
		m.groupCursor = 0
		return
	}

	position := m.groupPosition(m.groupSelection)
	if position < 0 {
		position = 0
		m.groupSelection = indices[position]
	}
	position += direction
	if position < 0 {
		position = 0
	}
	if position >= len(indices) {
		position = len(indices) - 1
	}
	m.groupCursor = position
	m.groupSelection = indices[position]
}

func (m *Model) selectedGroupIndex() int {
	indices := m.groupIndicesForCurrentView()
	if m.groupCursor < 0 || m.groupCursor >= len(indices) {
		m.normalizeGroupSelection()
		indices = m.groupIndicesForCurrentView()
	}
	if m.groupCursor < 0 || m.groupCursor >= len(indices) {
		return -1
	}
	m.groupSelection = indices[m.groupCursor]
	return m.groupSelection
}

func (m *Model) reorderGroup(direction int) error {
	indices := m.groupIndicesForCurrentView()
	if len(indices) < 2 {
		return nil
	}

	position := m.groupPosition(m.groupSelection)
	if position < 0 {
		position = 0
		m.groupSelection = indices[position]
	}
	targetPosition := position + direction
	if targetPosition < 0 || targetPosition >= len(indices) {
		return nil
	}

	currentIndex := indices[position]
	targetIndex := indices[targetPosition]
	m.groups[currentIndex], m.groups[targetIndex] = m.groups[targetIndex], m.groups[currentIndex]
	if m.groupSelection == currentIndex {
		m.groupSelection = targetIndex
	} else if m.groupSelection == targetIndex {
		m.groupSelection = currentIndex
	}
	if m.activeGroup == currentIndex {
		m.activeGroup = targetIndex
	} else if m.activeGroup == targetIndex {
		m.activeGroup = currentIndex
	}
	m.groupCursor = targetPosition

	return m.save()
}

func (m *Model) deleteGroupAt(groupIndex int) error {
	if groupIndex < 0 || groupIndex >= len(m.groups) {
		m.deleteGroup = -1
		return nil
	}

	activeWasDeleted := m.activeGroup == groupIndex
	selectionWasDeleted := m.groupSelection == groupIndex
	if len(m.groups) == 1 {
		// Keep the persistence invariant that there is always a usable default
		// group, while making the destructive part explicit in the prompt.
		m.groups[0] = task.Group{Name: task.DefaultGroupName, Tasks: []task.Task{}}
		m.activeGroup = 0
		m.groupSelection = 0
	} else {
		m.groups = append(m.groups[:groupIndex], m.groups[groupIndex+1:]...)
		switch {
		case m.activeGroup > groupIndex:
			m.activeGroup--
		}
		if activeWasDeleted {
			m.activeGroup = -1
			m.selectEnabledGroup(groupIndex)
		}
		if m.groupSelection > groupIndex {
			m.groupSelection--
		} else if selectionWasDeleted {
			if groupIndex >= len(m.groups) {
				m.groupSelection = len(m.groups) - 1
			} else {
				m.groupSelection = groupIndex
			}
		}
	}

	m.deleteGroup = -1
	m.ensureTaskGroup()
	m.normalizeGroupSelection()
	return m.save()
}

func (m Model) save() error {
	return task.SaveGroups(m.groups, m.path)
}

func groupTaskCounts(group task.Group) (total, done, open int) {
	total = len(group.Tasks)
	for _, currentTask := range group.Tasks {
		if currentTask.Done {
			done++
		}
	}
	return total, done, total - done
}

func groupCountLabel(count int, singular string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %ss", count, singular)
}

func (m Model) groupOverview() string {
	enabled := 0
	disabled := 0
	totalTasks := 0
	doneTasks := 0
	openTasks := 0
	for _, group := range m.groups {
		if group.Disabled {
			disabled++
		} else {
			enabled++
		}
		total, done, open := groupTaskCounts(group)
		totalTasks += total
		doneTasks += done
		openTasks += open
	}

	return groupMetaStyle.Render(fmt.Sprintf(
		"%s  ·  %s enabled  ·  %s disabled  ·  %s  ·  %s done  ·  %s open",
		groupCountLabel(len(m.groups), "group"),
		groupCountLabel(enabled, "group"),
		groupCountLabel(disabled, "group"),
		groupCountLabel(totalTasks, "task"),
		groupCountLabel(doneTasks, "task"),
		groupCountLabel(openTasks, "task"),
	))
}

func (m Model) groupNavigation() string {
	items := make([]string, 0, len(m.groups)+1)
	if m.allGroups {
		items = append(items, selectedGroupStyle.Render("[All]"))
	} else {
		items = append(items, navigationStyle.Render("All"))
	}

	for _, groupIndex := range m.enabledGroupIndices() {
		group := m.groups[groupIndex]
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

func (m Model) writeGroup(b *strings.Builder, groupIndex int, position int, nameWidth, openWidth, doneWidth int) {
	group := m.groups[groupIndex]
	cursor := "  "
	if position == m.groupCursor {
		cursor = cursorStyle.Render("> ")
	}

	name := group.Name
	if position == m.groupCursor {
		name = selectedGroupStyle.Render(name)
	}
	namePadding := strings.Repeat(" ", max(0, nameWidth-lipgloss.Width(group.Name)))
	_, done, open := groupTaskCounts(group)
	statusText := "enabled"
	statusStyle := groupEnabledStyle
	if groupIndex == m.activeGroup && !group.Disabled {
		statusText = "current"
		statusStyle = selectedGroupStyle
	}
	if group.Disabled {
		statusText = "(disabled)"
		statusStyle = disabledStyle
	}
	fmt.Fprintf(
		b,
		"%s%s%s  %*d open  %*d done  %s\n",
		cursor,
		name,
		namePadding,
		openWidth,
		open,
		doneWidth,
		done,
		statusStyle.Render(statusText),
	)
}

func (m Model) renderGroupsView(b *strings.Builder) {
	indices := m.groupIndicesForCurrentView()
	if len(indices) == 0 {
		if len(m.groups) == 0 {
			b.WriteString(emptyTaskStyle.Render("No groups yet. Press n to create one."))
		} else {
			b.WriteString(emptyTaskStyle.Render("All groups are disabled. Press h to show disabled groups."))
		}
		b.WriteString("\n")
		return
	}

	nameWidth := 0
	openWidth := 1
	doneWidth := 1
	for _, groupIndex := range indices {
		nameWidth = max(nameWidth, lipgloss.Width(m.groups[groupIndex].Name))
		_, done, open := groupTaskCounts(m.groups[groupIndex])
		openWidth = max(openWidth, len(strconv.Itoa(open)))
		doneWidth = max(doneWidth, len(strconv.Itoa(done)))
	}
	for position, groupIndex := range indices {
		m.writeGroup(b, groupIndex, position, nameWidth, openWidth, doneWidth)
	}
}

func (m Model) renderTasksView(b *strings.Builder) {
	visible := m.visibleTasks()
	enabled := m.enabledGroupIndices()
	if len(enabled) == 0 {
		b.WriteString(emptyTaskStyle.Render("No enabled groups. Open Groups view (v), then press h to show disabled groups."))
		b.WriteString("\n")
		return
	}

	if m.allGroups {
		position := 0
		for enabledPosition, groupIndex := range enabled {
			group := m.groups[groupIndex]
			b.WriteString(groupHeaderStyle.Render(group.Name))
			b.WriteString("\n")

			groupTasks := m.visibleTasksForGroup(groupIndex)
			if len(groupTasks) == 0 {
				b.WriteString(emptyTaskStyle.Render("  No tasks yet."))
				b.WriteString("\n")
			} else {
				for _, ref := range groupTasks {
					m.writeTask(b, ref, position)
					position++
				}
			}

			if enabledPosition < len(enabled)-1 {
				b.WriteString("\n")
			}
		}
		return
	}

	if m.activeGroup < 0 || m.activeGroup >= len(m.groups) || m.groups[m.activeGroup].Disabled {
		b.WriteString(emptyTaskStyle.Render("No enabled group selected."))
		b.WriteString("\n")
		return
	}

	b.WriteString(groupHeaderStyle.Render(m.groups[m.activeGroup].Name))
	b.WriteString("\n")

	if len(visible) == 0 {
		b.WriteString(emptyTaskStyle.Render("No tasks yet."))
		b.WriteString("\n")
	} else {
		for position, ref := range visible {
			m.writeTask(b, ref, position)
		}
	}
}

func (m Model) deleteConfirmation() string {
	if m.deleteGroup < 0 || m.deleteGroup >= len(m.groups) {
		return ""
	}
	group := m.groups[m.deleteGroup]
	return fmt.Sprintf(
		"\nDelete group %q with %d tasks? Last group deletion resets to an empty %s. y/enter: confirm  n/esc: cancel",
		group.Name,
		len(group.Tasks),
		task.DefaultGroupName,
	)
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Tasky"))
	b.WriteString("\n")
	if m.view == groupsView {
		b.WriteString(groupHeaderStyle.Render("Groups"))
		b.WriteString("\n")
		b.WriteString(m.groupOverview())
		b.WriteString("\n\n")
		m.renderGroupsView(&b)
	} else {
		b.WriteString(groupHeaderStyle.Render("Tasks"))
		b.WriteString("\n")
		b.WriteString(m.groupNavigation())
		b.WriteString("\n\n")
		m.renderTasksView(&b)
	}

	if m.confirmDelete {
		b.WriteString(helpStyle.Render(m.deleteConfirmation()))
	} else if m.creatingGroup {
		fmt.Fprintf(&b, "\n%s\n", m.input.View())
		b.WriteString(helpStyle.Render("enter: create group  esc: cancel"))
	} else if m.creating || m.editing {
		fmt.Fprintf(&b, "\n%s\n", m.input.View())
		b.WriteString(helpStyle.Render("enter: save  esc: cancel"))
	} else if m.editingGroup {
		fmt.Fprintf(&b, "\n%s\n", m.input.View())
		b.WriteString(helpStyle.Render("enter: rename group  esc: cancel"))
	} else if m.view == groupsView {
		hideLabel := "show disabled"
		if m.showDisabled {
			hideLabel = "hide disabled"
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf(
			"v: tasks  up/down: select  n/c/g: new  e: rename  d: delete  alt+up/down: reorder  enter/space: toggle  h: %s  q: quit",
			hideLabel,
		)))
	} else {
		hideLabel := "hide done"
		if !m.showDone {
			hideLabel = "show done"
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf(
			"v: groups view  left/right/tab: switch group  a: all groups  c/g: new group  up/down: move  alt+up/down: reorder  enter/space: toggle done  n: new  e: edit  d: delete  h: %s  q: quit",
			hideLabel,
		)))
	}

	return b.String()
}
