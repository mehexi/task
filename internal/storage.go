package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Priority string

const (
	Low    Priority = "LOW"
	Medium Priority = "MEDIUM"
	High   Priority = "HIGH"
)

type SourceInfo struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Original string `json:"original"`
	Kind     string `json:"kind"`
}

type Task struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Done      bool        `json:"done"`
	Priority  Priority    `json:"priority"`
	ProjectID string      `json:"project_id"`
	ParentID  string      `json:"parent_id,omitempty"`
	DueDate   string      `json:"due_date"`
	Notes     string      `json:"notes"`
	CreatedAt time.Time   `json:"created_at"`
	Source    *SourceInfo `json:"source,omitempty"`
}

type Project struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	CodeDir  string `json:"code_dir,omitempty"`
}

type taskData struct {
	Projects []Project `json:"projects"`
	Tasks    []Task    `json:"tasks"`
}

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "oc-tasks")
}

func ensureDir() error {
	dir := dataDir()
	if dir == "" {
		return os.ErrNotExist
	}
	return os.MkdirAll(dir, 0755)
}

func dataFile() string {
	return filepath.Join(dataDir(), "tasks.json")
}

func LoadTasks() ([]Project, []Task, error) {
	path := dataFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, []Task{}, nil
		}
		return nil, nil, err
	}
	var td taskData
	if err := json.Unmarshal(data, &td); err != nil {
		return nil, nil, err
	}
	if td.Projects == nil {
		td.Projects = []Project{}
	}
	if td.Tasks == nil {
		td.Tasks = []Task{}
	}
	return td.Projects, td.Tasks, nil
}

func SaveTasks(projects []Project, tasks []Task) error {
	if err := ensureDir(); err != nil {
		return err
	}
	td := taskData{Projects: projects, Tasks: tasks}
	data, err := json.MarshalIndent(td, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFile(), data, 0644)
}

// TODO: add migration support for schema changes
// DONE: json fields use omitempty for backwards compat
