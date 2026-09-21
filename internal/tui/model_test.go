package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"tasky/internal/task"
)

// keyRune and keyType keep these tests close to the messages Bubble Tea sends
// to Update, while making the individual scenarios easy to read.
func keyRune(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func keyType(key tea.KeyType, alt bool) tea.KeyMsg {
	return tea.KeyMsg{Type: key, Alt: alt}
}

func updateKey(t *testing.T, model *Model, msg tea.KeyMsg) {
	t.Helper()
	_, _ = model.Update(msg)
	// Commands are intentionally not executed here.  Persistence and all
	// state transitions happen synchronously in Model.Update; text-input blink
	// commands are incidental to these behavior tests.
}

func enterText(t *testing.T, model *Model, value string) {
	t.Helper()
	model.input.SetValue(value)
	updateKey(t, model, keyType(tea.KeyEnter, false))
}

func groupsFromDisk(t *testing.T, path string) []task.Group {
	t.Helper()
	groups, err := task.LoadGroups(path)
	if err != nil {
		t.Fatalf("LoadGroups(%q) error = %v", path, err)
	}
	return groups
}

func requireContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("View() does not contain %q:\n%s", want, got)
	}
}

func requireNotContains(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Fatalf("View() unexpectedly contains %q:\n%s", unwanted, got)
	}
}

func TestTasksViewTaskWorkflowAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{
			Name: "Work",
			Tasks: []task.Task{
				{Text: "ship release"},
				{Text: "file report", Done: true},
			},
		},
		{Name: "Personal", Tasks: []task.Task{{Text: "call home"}}},
	}, path)

	// Existing task view starts with all tasks visible and still exposes the
	// original task navigation and filtering behavior.
	view := model.View()
	requireContains(t, view, "ship release")
	requireContains(t, view, "file report")

	updateKey(t, &model, keyRune("h"))
	view = model.View()
	requireContains(t, view, "ship release")
	requireNotContains(t, view, "file report")

	// Create a task in the active group and verify it survives a real save.
	updateKey(t, &model, keyRune("n"))
	enterText(t, &model, "book team review")
	if got := model.groups[0].Tasks[len(model.groups[0].Tasks)-1].Text; got != "book team review" {
		t.Fatalf("created task text = %q, want %q", got, "book team review")
	}
	loaded := groupsFromDisk(t, path)
	if got := loaded[0].Tasks[len(loaded[0].Tasks)-1].Text; got != "book team review" {
		t.Fatalf("saved created task text = %q, want %q", got, "book team review")
	}

	// Edit the first task through the same input path used by the UI.
	updateKey(t, &model, keyRune("h")) // show completed tasks again
	model.cursor = 0
	updateKey(t, &model, keyRune("e"))
	if !model.editing {
		t.Fatal("e did not enter task editing mode")
	}
	enterText(t, &model, "ship release today")
	if got := model.groups[0].Tasks[0].Text; got != "ship release today" {
		t.Fatalf("edited task text = %q, want %q", got, "ship release today")
	}

	// Toggle completion, then hide completed tasks again.  The edit and the
	// completion flag should both be persisted without changing other tasks.
	model.cursor = 0
	updateKey(t, &model, keyType(tea.KeyEnter, false))
	if !model.groups[0].Tasks[0].Done {
		t.Fatal("enter did not mark the selected task done")
	}
	updateKey(t, &model, keyRune("h"))
	requireNotContains(t, model.View(), "ship release today")
	requireContains(t, model.View(), "book team review")

	loaded = groupsFromDisk(t, path)
	if !loaded[0].Tasks[0].Done || loaded[0].Tasks[0].Text != "ship release today" {
		t.Fatalf("saved edited task = %#v, want done edited task", loaded[0].Tasks[0])
	}
}

func TestDisabledGroupsStayOutOfTasksUntilReenabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "Inbox", Tasks: []task.Task{{Text: "visible task"}}},
		{Name: "Archive", Disabled: true, Tasks: []task.Task{{Text: "archived task"}}},
	}, path)

	view := model.View()
	requireContains(t, view, "visible task")
	requireNotContains(t, view, "Archive")
	requireNotContains(t, view, "archived task")
	updateKey(t, &model, keyRune("a"))
	view = model.View()
	requireNotContains(t, view, "Archive")
	requireNotContains(t, view, "archived task")
	updateKey(t, &model, keyRune("a"))

	// The groups view defaults to enabled groups, and h reveals disabled ones.
	updateKey(t, &model, keyRune("v"))
	requireNotContains(t, model.View(), "Archive")
	updateKey(t, &model, keyRune("h"))
	requireContains(t, model.View(), "Archive")

	// Select Archive and enable it.  Its task data must be unchanged.
	updateKey(t, &model, keyType(tea.KeyDown, false))
	updateKey(t, &model, keyType(tea.KeyEnter, false))
	if model.groups[1].Disabled {
		t.Fatal("enter did not re-enable the selected disabled group")
	}
	if got := model.groups[1].Tasks[0].Text; got != "archived task" {
		t.Fatalf("re-enabled group task = %q, want %q", got, "archived task")
	}

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("a")) // inspect the re-enabled group in all-groups mode
	requireContains(t, model.View(), "Archive")
	requireContains(t, model.View(), "archived task")

	loaded := groupsFromDisk(t, path)
	if loaded[1].Disabled || len(loaded[1].Tasks) != 1 || loaded[1].Tasks[0].Text != "archived task" {
		t.Fatalf("saved re-enabled group = %#v, want enabled group with original task", loaded[1])
	}
}

func TestDisablingActiveGroupFallsBackToEnabledGroup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "Today", Tasks: []task.Task{{Text: "today task"}}},
		{Name: "Later", Tasks: []task.Task{{Text: "later task"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyType(tea.KeyEnter, false)) // disable active Today

	if !model.groups[0].Disabled {
		t.Fatal("enter did not disable the active group")
	}
	if model.activeGroup != 1 {
		t.Fatalf("activeGroup = %d after disabling group 0, want fallback group 1", model.activeGroup)
	}
	updateKey(t, &model, keyRune("v"))
	requireNotContains(t, model.View(), "today task")
	requireContains(t, model.View(), "later task")

	loaded := groupsFromDisk(t, path)
	if !loaded[0].Disabled || loaded[0].Tasks[0].Text != "today task" {
		t.Fatalf("saved disabled group = %#v, want disabled group retaining its task", loaded[0])
	}
}

func TestAllDisabledGroupsRenderAnEmptyTasksView(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "One", Tasks: []task.Task{{Text: "one task"}}},
		{Name: "Two", Tasks: []task.Task{{Text: "two task"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyType(tea.KeyEnter, false)) // disable One
	updateKey(t, &model, keyType(tea.KeyEnter, false)) // selection follows enabled Two
	updateKey(t, &model, keyRune("v"))
	// Creating a task while every group is disabled is a safe no-op; the
	// disabled data must not be silently moved into a new or fallback group.
	before := len(model.groups[0].Tasks) + len(model.groups[1].Tasks)
	updateKey(t, &model, keyRune("n"))
	enterText(t, &model, "must not be created")

	for i, group := range model.groups {
		if !group.Disabled {
			t.Fatalf("group %d = %#v, want disabled", i, group)
		}
	}
	after := len(model.groups[0].Tasks) + len(model.groups[1].Tasks)
	if after != before {
		t.Fatalf("all-disabled task creation changed task count from %d to %d", before, after)
	}
	view := model.View() // Must remain safe even though no task group is enabled.
	requireNotContains(t, view, "one task")
	requireNotContains(t, view, "two task")
}

func TestStartupSkipsDisabledFirstGroup(t *testing.T) {
	model := NewModel([]task.Group{
		{Name: "Disabled first", Disabled: true, Tasks: []task.Task{{Text: "hidden first"}}},
		{Name: "Ready", Tasks: []task.Task{{Text: "ready task"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	if model.activeGroup != 1 {
		t.Fatalf("activeGroup at startup = %d, want first enabled group 1", model.activeGroup)
	}
	requireContains(t, model.View(), "ready task")
	requireNotContains(t, model.View(), "hidden first")
}

func TestGroupsViewRenameDeleteConfirmationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "Inbox", Tasks: []task.Task{{Text: "keep me"}}},
		{Name: "Someday", Tasks: []task.Task{{Text: "later"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("e"))
	enterText(t, &model, "Renamed inbox")
	if got := model.groups[0].Name; got != "Renamed inbox" {
		t.Fatalf("renamed group = %q, want %q", got, "Renamed inbox")
	}

	// A cancellation must leave the renamed group intact.
	updateKey(t, &model, keyRune("d"))
	updateKey(t, &model, keyRune("n"))
	if len(model.groups) != 2 || model.groups[0].Name != "Renamed inbox" {
		t.Fatalf("delete cancellation changed groups: %#v", model.groups)
	}

	// Confirming with enter removes only the selected group and preserves the
	// remaining group's tasks and its order.
	updateKey(t, &model, keyRune("d"))
	updateKey(t, &model, keyType(tea.KeyEnter, false))
	if len(model.groups) != 1 || model.groups[0].Name != "Someday" {
		t.Fatalf("confirmed deletion left groups = %#v, want Someday", model.groups)
	}
	if len(model.groups[0].Tasks) != 1 || model.groups[0].Tasks[0].Text != "later" {
		t.Fatalf("remaining group tasks = %#v, want later", model.groups[0].Tasks)
	}

	loaded := groupsFromDisk(t, path)
	if len(loaded) != 1 || loaded[0].Name != "Someday" || loaded[0].Tasks[0].Text != "later" {
		t.Fatalf("saved groups after delete = %#v", loaded)
	}
}

func TestDeletingLastGroupResetsEmptyGeneral(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "Only group", Tasks: []task.Task{{Text: "remove with group"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("d"))
	updateKey(t, &model, keyRune("y"))

	if len(model.groups) != 1 || model.groups[0].Name != task.DefaultGroupName {
		t.Fatalf("deleting last group left groups = %#v, want empty General", model.groups)
	}
	if model.groups[0].Disabled || len(model.groups[0].Tasks) != 0 {
		t.Fatalf("reset group = %#v, want enabled and empty", model.groups[0])
	}
	loaded := groupsFromDisk(t, path)
	if len(loaded) != 1 || loaded[0].Name != task.DefaultGroupName || loaded[0].Disabled || len(loaded[0].Tasks) != 0 {
		t.Fatalf("saved reset group = %#v, want enabled empty General", loaded)
	}
}

func TestGroupsViewCreatesGroupAndPersistsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "Inbox", Tasks: []task.Task{{Text: "keep"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("c"))
	enterText(t, &model, "Created in groups")

	var created *task.Group
	for i := range model.groups {
		if model.groups[i].Name == "Created in groups" {
			created = &model.groups[i]
			break
		}
	}
	if created == nil {
		t.Fatalf("groups after creation = %#v, want Created in groups", model.groups)
	}
	if created.Disabled || len(created.Tasks) != 0 {
		t.Fatalf("created group = %#v, want enabled and empty", *created)
	}
	requireContains(t, model.View(), "Created in groups")

	loaded := groupsFromDisk(t, path)
	found := false
	for _, group := range loaded {
		if group.Name == "Created in groups" {
			found = true
			if group.Disabled || len(group.Tasks) != 0 {
				t.Fatalf("saved created group = %#v, want enabled and empty", group)
			}
		}
	}
	if !found {
		t.Fatalf("saved groups = %#v, want created group", loaded)
	}
}

func TestGroupsViewShowsOverviewAndTaskBreakdown(t *testing.T) {
	model := NewModel([]task.Group{
		{
			Name: "Inbox",
			Tasks: []task.Task{
				{Text: "ship it"},
				{Text: "done", Done: true},
			},
		},
		{Name: "Someday", Disabled: true, Tasks: []task.Task{{Text: "later"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	updateKey(t, &model, keyRune("v"))
	view := model.View()
	requireContains(t, view, "2 groups")
	requireContains(t, view, "1 group enabled")
	requireContains(t, view, "1 group disabled")
	requireContains(t, view, "3 tasks")
	requireContains(t, view, "1 task done")
	requireContains(t, view, "2 tasks open")
	requireContains(t, view, "1 task open")
	requireContains(t, view, "current")
	requireNotContains(t, view, "Someday")

	updateKey(t, &model, keyRune("h"))
	view = model.View()
	requireContains(t, view, "Someday")
	requireContains(t, view, "(disabled)")
}

func TestCreatingGroupRecoversFromAllDisabledState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "One", Tasks: []task.Task{{Text: "one"}}},
		{Name: "Two", Tasks: []task.Task{{Text: "two"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyType(tea.KeyEnter, false))
	updateKey(t, &model, keyType(tea.KeyEnter, false))
	for _, group := range model.groups {
		if !group.Disabled {
			t.Fatalf("before recovery groups = %#v, want all disabled", model.groups)
		}
	}

	updateKey(t, &model, keyRune("c"))
	enterText(t, &model, "Recovered")
	var recovered *task.Group
	for i := range model.groups {
		if model.groups[i].Name == "Recovered" {
			recovered = &model.groups[i]
			break
		}
	}
	if recovered == nil || recovered.Disabled {
		t.Fatalf("groups after recovery = %#v, want enabled Recovered", model.groups)
	}

	updateKey(t, &model, keyRune("v"))
	requireContains(t, model.View(), "Recovered")
	updateKey(t, &model, keyRune("n"))
	enterText(t, &model, "recovered task")
	if len(recovered.Tasks) != 1 || recovered.Tasks[0].Text != "recovered task" {
		t.Fatalf("recovered group tasks = %#v, want recovered task", recovered.Tasks)
	}
	loaded := groupsFromDisk(t, path)
	for _, group := range loaded {
		if group.Name == "Recovered" {
			if len(group.Tasks) != 1 || group.Tasks[0].Text != "recovered task" {
				t.Fatalf("saved recovered group = %#v", group)
			}
			return
		}
	}
	t.Fatalf("saved groups = %#v, missing Recovered", loaded)
}

func TestTaskNavigationSkipsDisabledGroupsInBothDirections(t *testing.T) {
	model := NewModel([]task.Group{
		{Name: "First", Tasks: []task.Task{{Text: "first task"}}},
		{Name: "Disabled middle", Disabled: true, Tasks: []task.Task{{Text: "disabled task"}}},
		{Name: "Last", Tasks: []task.Task{{Text: "last task"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	updateKey(t, &model, keyType(tea.KeyRight, false))
	view := model.View()
	requireContains(t, view, "last task")
	requireNotContains(t, view, "first task")
	requireNotContains(t, view, "disabled task")
	requireNotContains(t, view, "Disabled middle")

	updateKey(t, &model, keyType(tea.KeyLeft, false))
	view = model.View()
	requireContains(t, view, "first task")
	requireNotContains(t, view, "last task")
	requireNotContains(t, view, "disabled task")
}

func TestGroupsSelectionDoesNotChangeTaskSelection(t *testing.T) {
	model := NewModel([]task.Group{
		{Name: "Alpha", Tasks: []task.Task{{Text: "alpha task"}}},
		{Name: "Beta", Tasks: []task.Task{{Text: "beta task"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	// Groups-view selection and reordering must not silently switch the task
	// group the user was working in.  The selected Groups row is Beta, while
	// the Tasks view should still return to Alpha.
	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyType(tea.KeyDown, false))
	updateKey(t, &model, keyType(tea.KeyUp, true))
	updateKey(t, &model, keyRune("v"))
	view := model.View()
	requireContains(t, view, "alpha task")
	requireNotContains(t, view, "beta task")
}

func TestAllGroupsModeSurvivesGroupsViewRoundTrip(t *testing.T) {
	model := NewModel([]task.Group{
		{Name: "Alpha", Tasks: []task.Task{{Text: "alpha task"}}},
		{Name: "Beta", Tasks: []task.Task{{Text: "beta task"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	updateKey(t, &model, keyRune("a"))
	view := model.View()
	requireContains(t, view, "alpha task")
	requireContains(t, view, "beta task")

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("v"))
	if !model.allGroups {
		t.Fatal("all-groups mode was lost after a Groups view round-trip")
	}
	view = model.View()
	requireContains(t, view, "alpha task")
	requireContains(t, view, "beta task")
}

func TestCtrlCQuitsDuringDeleteConfirmation(t *testing.T) {
	model := NewModel([]task.Group{
		{Name: "Inbox", Tasks: []task.Task{{Text: "keep"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("d"))
	if !model.confirmDelete {
		t.Fatal("d did not enter delete confirmation")
	}
	_, cmd := model.Update(keyType(tea.KeyCtrlC, false))
	if cmd == nil {
		t.Fatal("ctrl+c during delete confirmation returned no quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c command returned %T, want tea.QuitMsg", msg)
	}
}

func TestGroupsViewReordersVisibleGroupsAndPreservesSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	model := NewModel([]task.Group{
		{Name: "Alpha", Tasks: []task.Task{{Text: "alpha"}}},
		{Name: "Hidden middle", Disabled: true, Tasks: []task.Task{{Text: "hidden"}}},
		{Name: "Beta", Tasks: []task.Task{{Text: "beta"}}},
	}, path)

	updateKey(t, &model, keyRune("v"))
	// Disabled groups are filtered from the default list, so Down selects Beta
	// directly even though it is separated by Hidden middle in storage order.
	updateKey(t, &model, keyType(tea.KeyDown, false))
	updateKey(t, &model, keyType(tea.KeyUp, true))

	// The selected group's identity remains Beta after the filtered reorder.
	// Rename the selected row to prove selection did not jump to Alpha or the
	// hidden row while the underlying slice was rearranged.
	updateKey(t, &model, keyRune("e"))
	enterText(t, &model, "Beta renamed")
	var enabledNames []string
	for _, group := range model.groups {
		if !group.Disabled {
			enabledNames = append(enabledNames, group.Name)
		}
	}
	if len(enabledNames) != 2 || enabledNames[0] != "Beta renamed" || enabledNames[1] != "Alpha" {
		t.Fatalf("enabled group order after filtered reorder = %#v, want [Beta renamed Alpha]", enabledNames)
	}

	loaded := groupsFromDisk(t, path)
	var loadedEnabled []string
	for _, group := range loaded {
		if !group.Disabled {
			loadedEnabled = append(loadedEnabled, group.Name)
		}
	}
	if len(loadedEnabled) != 2 || loadedEnabled[0] != "Beta renamed" || loadedEnabled[1] != "Alpha" {
		t.Fatalf("saved enabled group order = %#v, want [Beta renamed Alpha]", loadedEnabled)
	}
	for _, group := range loaded {
		if group.Name == "Hidden middle" {
			if !group.Disabled || len(group.Tasks) != 1 || group.Tasks[0].Text != "hidden" {
				t.Fatalf("hidden group changed during reorder: %#v", group)
			}
		}
	}
}

func TestGroupsHiddenToggleDoesNotChangeTaskDoneFilter(t *testing.T) {
	model := NewModel([]task.Group{
		{Name: "Active", Tasks: []task.Task{{Text: "done task", Done: true}}},
		{Name: "Disabled", Disabled: true, Tasks: []task.Task{{Text: "hidden group task"}}},
	}, filepath.Join(t.TempDir(), "tasks.json"))

	if !model.showDone {
		t.Fatal("NewModel should show completed tasks by default")
	}
	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("h"))
	if !model.showDone {
		t.Fatal("groups h unexpectedly changed task completion filter")
	}
	updateKey(t, &model, keyRune("v"))
	updateKey(t, &model, keyRune("h"))
	requireNotContains(t, model.View(), "done task")
}
