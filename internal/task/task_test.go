package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSaveAndLoadGroupsRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.json")
	expected := []Group{
		{
			Name: "Inbox",
			Tasks: []Task{
				{Text: "first", Done: false},
				{Text: "finished", Done: true},
			},
		},
		{
			Name:     "Someday",
			Disabled: true,
			Tasks: []Task{
				{Text: "keep this task", Done: false},
			},
		},
		{
			Name:  "Later",
			Tasks: []Task{},
		},
	}

	if err := SaveGroups(expected, path); err != nil {
		t.Fatalf("SaveGroups() error = %v", err)
	}

	loaded, err := LoadGroups(path)
	if err != nil {
		t.Fatalf("LoadGroups() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, expected) {
		t.Fatalf("LoadGroups() = %#v, want %#v", loaded, expected)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	contents := string(data)
	if !strings.Contains(contents, `"disabled": true`) {
		t.Fatalf("saved JSON does not persist disabled groups: %s", contents)
	}
	if strings.Contains(contents, `"disabled": false`) {
		t.Fatalf("saved JSON should omit enabled disabled=false values: %s", contents)
	}
}

func TestLoadGroupsOlderGroupedFileDefaultsToEnabled(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.json")
	data := []byte(`{"groups":[
  {"name":"Work","tasks":[{"text":"ship release","done":false}]},
  {"name":"Personal","tasks":[{"text":"call home","done":true}]}
]}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := LoadGroups(path)
	if err != nil {
		t.Fatalf("LoadGroups() error = %v", err)
	}
	want := []Group{
		{Name: "Work", Tasks: []Task{{Text: "ship release", Done: false}}},
		{Name: "Personal", Tasks: []Task{{Text: "call home", Done: true}}},
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadGroups() = %#v, want %#v", loaded, want)
	}
	for _, group := range loaded {
		if group.Disabled {
			t.Fatalf("older grouped file marked %q disabled; missing field should default to enabled", group.Name)
		}
	}
}

func TestLoadGroupsLegacyFlatFileKeepsTasksInGeneral(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tasks.json")
	legacy := []Task{
		{Text: "first legacy task", Done: false},
		{Text: "second legacy task", Done: true},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := LoadGroups(path)
	if err != nil {
		t.Fatalf("LoadGroups() error = %v", err)
	}
	want := []Group{{Name: DefaultGroupName, Tasks: legacy}}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadGroups() = %#v, want %#v", loaded, want)
	}
	if loaded[0].Disabled {
		t.Fatalf("legacy flat file should load into an enabled %q group", DefaultGroupName)
	}
}
