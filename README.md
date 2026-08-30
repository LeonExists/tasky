```
███████   █████    █████   █     █  █     █
   █     █     █  █        █    █    █   █
   █     █     █  █        █   █      ███
   █     ███████   █████   ████        █
   █     █     █        █  █   █       █
   █     █     █   █████   █    █      █
```

A simple CLI written in Go to keep track of all your todos and sync them across your notes in Obsidian.

> **Status:** task persistence and the interactive TUI work; Obsidian sync is not built yet.

## Building

```
go build ./cmd/tasky
```

## Installing

To make the `tasky` command available in your terminal:

```
go install ./cmd/tasky
```

This builds the binary and drops it into your Go bin directory — `go env GOBIN` if set,
otherwise `bin` inside `go env GOPATH` (`%USERPROFILE%\go\bin` by default on Windows).
Make sure that directory is on your `PATH`:

```
# PowerShell (add to your $PROFILE to make it permanent)
$env:Path += ";$(go env GOPATH)\bin"
```

```
# bash/zsh (add to your shell rc file to make it permanent)
export PATH="$PATH:$(go env GOPATH)/bin"
```

Once that's done, `tasky` runs from any directory.

## Configuration

Tasky looks for a config file at `~/.tasky/config.json` (`%USERPROFILE%\.tasky\config.json`
on Windows). It's optional — with no config file, tasks are stored at `~/.tasky/tasks.json`
and logging is off. To customize either, create the file yourself:

```json
{
  "tasksPath": "C:/Users/you/.tasky/tasks.json",
  "logEnabled": true
}
```

- `tasksPath` — where your tasks are saved as JSON. Defaults to `~/.tasky/tasks.json`.
- `logEnabled` — whether Tasky prints log messages while running. Defaults to `false`.

## Usage

Run the built binary (or `go run ./cmd/tasky`) to open the TUI:

| Key            | Action                          |
| -------------- | -------------------------------- |
| `up`/`k`       | Move cursor up                   |
| `down`/`j`     | Move cursor down                 |
| `alt+up`       | Move the selected todo up        |
| `alt+down`     | Move the selected todo down      |
| `enter`/space  | Toggle the selected todo done    |
| `n`            | Create a new todo                |
| `d`            | Delete the selected todo         |
| `h`            | Toggle hiding completed todos    |
| `q`            | Quit                             |

## Roadmap

- [x] Task type with JSON persistence (`SaveAll`/`LoadAll`)
- [x] Interactive TUI (built on [Bubble Tea](https://github.com/charmbracelet/bubbletea))
- [ ] Sync tasks with Obsidian notes
