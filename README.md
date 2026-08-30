# tasky
A simple CLI written in Go to keep track of all your todos and sync them across your notes in Obsidian.

> **Status:** early scaffolding. Task creation and JSON persistence work; the TUI and Obsidian sync are not built yet.

## Building

```
go build ./cmd/tasky
```

## Configuration

Tasky reads a `config.json` file that points it at where your tasks are stored:

```json
{
  "tasksPath": "tasks.json"
}
```

## Roadmap

- [x] Task type with JSON persistence (`SaveAll`/`LoadAll`)
- [ ] Interactive TUI (built on [Bubble Tea](https://github.com/charmbracelet/bubbletea))
- [ ] Sync tasks with Obsidian notes
