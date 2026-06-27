# oc-tasks

A terminal-based task manager with TUI, CLI, code comment scanning, and gamification.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Three-pane TUI** — projects, tasks, and detail view in one terminal window
- **Full CLI** — add, list, done, delete, load, and manage tasks non-interactively
- **Code comment scanner** — auto-discovers `TODO`, `FIXME`, `HACK`, `NOTE`, `XXX` comments in your codebase and creates tasks from them
- **Subtask support** — nest tasks with parent-child relationships
- **Gamification** — XP, levels, streaks, and achievements to keep you motivated
- **Priorities** — Low / Medium / High with color-coded badges
- **Due dates** — set dates on tasks; overdue tasks earn bonus XP
- **Filtering** — filter tasks by text in the TUI or by project/status in CLI
- **JSON export** — all CLI commands output JSON for easy piping and scripting
- **Auto-reload** — detects external file changes and reloads
- **Theme support** — integrates with [Omarchy](https://github.com/anomalyco/omarchy) colors

## Installation

### Go install

```bash
go install github.com/mehexi/task/cmd/oc-tasks@latest
```

### Download a release binary

Grab the latest binary for your platform from the [releases page](https://github.com/mehexi/task/releases), rename it to `oc-tasks`, make it executable, and put it in your `$PATH`:

```bash
chmod +x oc-tasks
sudo mv oc-tasks /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/mehexi/task.git
cd task
go build -o oc-tasks ./cmd/oc-tasks/
sudo mv oc-tasks /usr/local/bin/
```

## Usage

### TUI (Interactive)

```bash
oc-tasks
```

Key bindings:

| Key | Action |
|-----|--------|
| `j`/`k` or `↑`/`↓` | Navigate |
| `Tab` / `1` `2` `3` | Switch pane |
| `Space` | Toggle task done |
| `n` | New task / project |
| `a` | New subtask |
| `d` | Delete task / project |
| `p` | Cycle priority |
| `e` | Edit notes |
| `D` | Set due date |
| `/` | Filter tasks |
| `R` | Rescan code comments |
| `q` / `Ctrl+C` | Quit |

### CLI

```bash
# Add a task
oc-tasks add "Write documentation" --project docs --priority HIGH --due 2026-07-15 --notes "focus on API"

# List tasks
oc-tasks list
oc-tasks list --project docs --status todo

# Mark done
oc-tasks done <task-id>

# Delete
oc-tasks delete <task-id>

# Manage projects
oc-tasks project list
oc-tasks project create "MyProject"

# Bulk load from JSON
oc-tasks load tasks.json

# Show subtasks
oc-tasks subtasks <task-id>

# Scan code comments
oc-tasks scan /path/to/project

# Launch TUI explicitly (e.g., from a script)
oc-tasks tui
```

### Load file format (JSON)

```json
[
  {"title": "Fix login bug", "project": "Backend", "priority": "HIGH", "due": "2026-07-01", "notes": "Session timeout issue"},
  {"title": "Design homepage", "project": "Frontend", "priority": "MEDIUM"},
  {"title": "Write tests", "project": "Backend", "priority": "LOW"}
]
```

### Scan output

```bash
oc-tasks scan .
```

Scans all recognized source files for `TODO`, `FIXME`, `HACK`, `NOTE`, `XXX` comments, respects `.gitignore`, and outputs them as JSON.

## Data

Tasks are stored in `~/.local/share/oc-tasks/tasks.json`. External modifications are auto-detected and reloaded.

## Gamification

| Action | XP |
|--------|----|
| Complete a Low task | 10 |
| Complete a Medium task | 15 |
| Complete a High task | 20 |
| Complete an overdue task | +5 |

Achievements: First Step, Getting Started, Completionist, On Fire, Unstoppable, Prioritizer, Clean Sweep, Speed Run, Early Bird, Organized

## Contributing

Contributions welcome! Please open an issue or PR on [GitHub](https://github.com/mehexi/task).

## License

MIT
