package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func isInteractive() bool {
	fi, _ := os.Stdout.Stat()
	return fi != nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func Run(args []string) error {
	LoadThemeColors()
	if len(args) == 0 {
		if isInteractive() {
			return runTUI()
		}
		return runList(nil)
	}
	switch args[0] {
	case "tui":
		return runTUI()
	case "add":
		return runAdd(args[1:])
	case "list":
		return runList(args[1:])
	case "done":
		return runDone(args[1:])
	case "delete":
		return runDelete(args[1:])
	case "project":
		return runProject(args[1:])
	case "load":
		return runLoad(args[1:])
	case "subtasks":
		return runSubtasks(args[1:])
	case "scan":
		return runScan(args[1:])
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		if !strings.HasPrefix(args[0], "-") {
			return fmt.Errorf("unknown command: %s\nRun 'oc-tasks help' for usage", args[0])
		}
		return runAdd(args)
	}
}

func printHelp() {
	fmt.Print(`oc-tasks — terminal TUI task manager

Usage:
  oc-tasks                  Launch TUI (interactive terminal only)
  oc-tasks tui              Launch TUI (always)
  oc-tasks add <title>      Add a task
  oc-tasks list             List tasks (JSON)
  oc-tasks done <id>        Mark task done
  oc-tasks delete <id>      Delete a task
  oc-tasks project list     List projects (JSON)
  oc-tasks project create <name>  Create a project
  oc-tasks load <file>      Load tasks from JSON file
  oc-tasks subtasks <id>    List subtasks of a task (JSON)
  oc-tasks scan [dir]       Scan directory for code comments (JSON)
  oc-tasks help             Show this help

Add flags:
  --title "Task title"      Task title (or use positional arg)
  --project "Project name"  Project name (created if missing)
  --priority LOW|MEDIUM|HIGH  Priority (default: LOW)
  --parent <task-id>        Parent task ID (makes this a subtask)
  --due YYYY-MM-DD          Due date
  --notes "Notes"           Task notes

List flags:
  --project "Name"          Filter by project name
  --status todo|done        Filter by status

Subtasks flags:
  --status todo|done        Filter by status

Load file format (JSON array):
  [{"title":"...", "project":"...", "priority":"HIGH", ...}]
`)
}

func runTUI() error {
	m := New()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func runAdd(args []string) error {
	// Reorder args so flags (and their values) come before positional args.
	// Go's flag.FlagSet.Parse stops at the first non-flag arg.
	var reordered, positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			reordered = append(reordered, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				reordered = append(reordered, args[i+1])
				i++
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	title := fs.String("title", "", "Task title")
	project := fs.String("project", "", "Project name")
	priority := fs.String("priority", "LOW", "Priority (LOW, MEDIUM, HIGH)")
	due := fs.String("due", "", "Due date (YYYY-MM-DD)")
	notes := fs.String("notes", "", "Task notes")
	parent := fs.String("parent", "", "Parent task ID")
	if err := fs.Parse(append(reordered, positional...)); err != nil {
		return err
	}
	t := *title
	if t == "" && fs.NArg() > 0 {
		t = fs.Arg(0)
	}
	t = strings.TrimSpace(t)
	if t == "" {
		return fmt.Errorf("title is required")
	}
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	projName := *project
	if projName == "" && len(projects) > 0 {
		projName = projects[0].Name
	}
	var projID string
	for _, p := range projects {
		if strings.EqualFold(p.Name, projName) {
			projID = p.ID
			break
		}
	}
	if projID == "" {
		p := Project{
			ID:    generateID(),
			Name:  projName,
			Color: "#7aa2f7",
		}
		projects = append(projects, p)
		projID = p.ID
	}
	pr := Low
	switch strings.ToUpper(*priority) {
	case "HIGH":
		pr = High
	case "MEDIUM":
		pr = Medium
	}
	task := Task{
		ID:        generateID(),
		Title:     t,
		Done:      false,
		Priority:  pr,
		ProjectID: projID,
		ParentID:  *parent,
		DueDate:   *due,
		Notes:     *notes,
		CreatedAt: time.Now(),
	}
	tasks = append(tasks, task)
	if err := SaveTasks(projects, tasks); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(task)
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	project := fs.String("project", "", "Filter by project name")
	status := fs.String("status", "", "Filter by status (todo, done)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	projNames := make(map[string]string)
	for _, p := range projects {
		projNames[p.ID] = p.Name
	}
	var projID string
	if *project != "" {
		for _, p := range projects {
			if strings.EqualFold(p.Name, *project) {
				projID = p.ID
				break
			}
		}
	}
	var filtered []Task
	for _, t := range tasks {
		if projID != "" && t.ProjectID != projID {
			continue
		}
		if *status == "todo" && t.Done {
			continue
		}
		if *status == "done" && !t.Done {
			continue
		}
		filtered = append(filtered, t)
	}
	if filtered == nil {
		filtered = []Task{}
	}
	return json.NewEncoder(os.Stdout).Encode(filtered)
}

func runDone(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}
	id := args[0]
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	found := false
	for i, t := range tasks {
		if t.ID == id {
			tasks[i].Done = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("task not found: %s", id)
	}
	if err := SaveTasks(projects, tasks); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "done", "id": id})
}

func runDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}
	id := args[0]
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	found := false
	var updated []Task
	for _, t := range tasks {
		if t.ID == id {
			found = true
		} else {
			updated = append(updated, t)
		}
	}
	if !found {
		return fmt.Errorf("task not found: %s", id)
	}
	if err := SaveTasks(projects, updated); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "deleted", "id": id})
}

func runProject(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("subcommand required: list, create")
	}
	switch args[0] {
	case "list":
		return runProjectList(args[1:])
	case "create":
		return runProjectCreate(args[1:])
	default:
		return fmt.Errorf("unknown project subcommand: %s\nUsage: oc-tasks project list|create", args[0])
	}
}

func runProjectList(_ []string) error {
	projects, _, err := LoadTasks()
	if err != nil {
		return err
	}
	if projects == nil {
		projects = []Project{}
	}
	return json.NewEncoder(os.Stdout).Encode(projects)
}

func runProjectCreate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project name is required")
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"id": p.ID, "name": p.Name, "status": "exists",
			})
		}
	}
	p := Project{
		ID:    generateID(),
		Name:  name,
		Color: "#7aa2f7",
	}
	projects = append(projects, p)
	if err := SaveTasks(projects, tasks); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"id": p.ID, "name": p.Name, "status": "created",
	})
}

type taskInput struct {
	Title    string   `json:"title"`
	Project  string   `json:"project"`
	Priority Priority `json:"priority"`
	Due      string   `json:"due"`
	Notes    string   `json:"notes"`
}

func runLoad(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("file path is required")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	var inputs []taskInput
	if err := json.Unmarshal(data, &inputs); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	if len(inputs) == 0 {
		return fmt.Errorf("no tasks found in file")
	}
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	projCache := make(map[string]string)
	for _, p := range projects {
		projCache[strings.ToLower(p.Name)] = p.ID
	}
	var created []Task
	for _, in := range inputs {
		title := strings.TrimSpace(in.Title)
		if title == "" {
			continue
		}
		projName := in.Project
		if projName == "" && len(projects) > 0 {
			projName = projects[0].Name
		}
		projID, ok := projCache[strings.ToLower(projName)]
		if !ok {
			p := Project{
				ID:    generateID(),
				Name:  projName,
				Color: "#7aa2f7",
			}
			projects = append(projects, p)
			projID = p.ID
			projCache[strings.ToLower(projName)] = projID
		}
		pr := in.Priority
		if pr == "" {
			pr = Low
		}
		task := Task{
			ID:        generateID(),
			Title:     title,
			Done:      false,
			Priority:  pr,
			ProjectID: projID,
			DueDate:   in.Due,
			Notes:     in.Notes,
			CreatedAt: time.Now(),
		}
		tasks = append(tasks, task)
		created = append(created, task)
	}
	if len(created) == 0 {
		return fmt.Errorf("no valid tasks to create")
	}
	if err := SaveTasks(projects, tasks); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(created)
}

func runScan(args []string) error {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}
	items, err := FindComments(rootDir)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", rootDir, err)
	}
	if items == nil {
		items = []CommentItem{}
	}
	return json.NewEncoder(os.Stdout).Encode(items)
}

func runSubtasks(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}
	id := args[0]
	projects, tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	_ = projects
	projNames := make(map[string]string)
	for _, p := range projects {
		projNames[p.ID] = p.Name
	}

	fs := flag.NewFlagSet("subtasks", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status (todo, done)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	var result []Task
	for _, t := range tasks {
		if t.ParentID != id {
			continue
		}
		if *status == "todo" && t.Done {
			continue
		}
		if *status == "done" && !t.Done {
			continue
		}
		result = append(result, t)
	}
	if result == nil {
		result = []Task{}
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
