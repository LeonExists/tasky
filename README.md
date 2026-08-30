# tasky
A simple CLI written in Go to keep track of all your todos and sync them across your notes in Obsidian.

> **Status:** task persistence and the interactive TUI work; Obsidian sync is not built yet.

## Building

```
go build ./cmd/tasky
```

## Configuration

Tasky reads a `config.json` file that points it at where your tasks are stored and whether logging is enabled:

```json
{
  "tasksPath": "tasks.json",
  "logEnabled": true
}
```

## Usage

Run the built binary (or `go run ./cmd/tasky`) to open the TUI:

| Key           | Action                          |
| ------------- | -------------------------------- |
| `up`/`k`      | Move cursor up                   |
| `down`/`j`    | Move cursor down                 |
| `enter`/space | Toggle the selected todo done    |
| `n`           | Create a new todo                |
| `d`           | Delete the selected todo         |
| `h`           | Toggle hiding completed todos    |
| `q`           | Quit                             |

## Roadmap

- [x] Task type with JSON persistence (`SaveAll`/`LoadAll`)
- [x] Interactive TUI (built on [Bubble Tea](https://github.com/charmbracelet/bubbletea))
- [ ] Sync tasks with Obsidian notes
