package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withTempHome(t *testing.T, fn func()) {
	t.Helper()
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserHome := ""
	var ok bool
	// Some systems use USERPROFILE (Windows), but we're on Linux
	os.Setenv("HOME", tmpHome)
	if oldUserHome, ok = os.LookupEnv("XDG_DATA_HOME"); ok {
		os.Unsetenv("XDG_DATA_HOME")
	}
	defer func() {
		os.Setenv("HOME", oldHome)
		if ok {
			os.Setenv("XDG_DATA_HOME", oldUserHome)
		}
	}()
	fn()
}

func TestSaveAndLoadTasks(t *testing.T) {
	withTempHome(t, func() {
		projects := []Project{
			{ID: "proj1", Name: "Test Project", Color: "#7aa2f7"},
		}
		tasks := []Task{
			{
				ID:        "task1",
				Title:     "Test Task",
				Done:      false,
				Priority:  High,
				ProjectID: "proj1",
				CreatedAt: time.Now(),
			},
		}
		gam := Gamification{
			XP:    50,
			Level: 2,
		}

		if err := SaveTasks(projects, tasks, gam); err != nil {
			t.Fatalf("SaveTasks failed: %v", err)
		}

		gotProjects, gotTasks, gotGam, err := LoadTasks()
		if err != nil {
			t.Fatalf("LoadTasks failed: %v", err)
		}

		if len(gotProjects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(gotProjects))
		}
		if gotProjects[0].Name != "Test Project" {
			t.Errorf("project name = %q, want %q", gotProjects[0].Name, "Test Project")
		}

		if len(gotTasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(gotTasks))
		}
		if gotTasks[0].Title != "Test Task" {
			t.Errorf("task title = %q, want %q", gotTasks[0].Title, "Test Task")
		}
		if gotTasks[0].Priority != High {
			t.Errorf("task priority = %q, want %q", gotTasks[0].Priority, High)
		}

		if gotGam.XP != 50 {
			t.Errorf("gam XP = %d, want %d", gotGam.XP, 50)
		}
		if gotGam.Level != 2 {
			t.Errorf("gam Level = %d, want %d", gotGam.Level, 2)
		}
	})
}

func TestLoadTasks_FileNotFound(t *testing.T) {
	withTempHome(t, func() {
		projects, tasks, gam, err := LoadTasks()
		if err != nil {
			t.Fatalf("LoadTasks when no file exists: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("expected 0 projects, got %d", len(projects))
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
		if gam.Level != 0 {
			t.Errorf("expected Level 0 for new gam, got %d", gam.Level)
		}
	})
}

func TestLoadTasks_InvalidJSON(t *testing.T) {
	withTempHome(t, func() {
		dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "oc-tasks")
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "tasks.json"), []byte("{invalid json}"), 0644)

		_, _, _, err := LoadTasks()
		if err == nil {
			t.Errorf("expected error for invalid JSON")
		}
	})
}

func TestSaveTasks_EmptyStates(t *testing.T) {
	withTempHome(t, func() {
		if err := SaveTasks([]Project{}, []Task{}, Gamification{}); err != nil {
			t.Fatalf("SaveTasks with empty data: %v", err)
		}

		projects, tasks, gam, err := LoadTasks()
		if err != nil {
			t.Fatalf("LoadTasks: %v", err)
		}
		if projects == nil {
			t.Errorf("projects should not be nil")
		}
		if tasks == nil {
			t.Errorf("tasks should not be nil")
		}
		if gam.Achievements == nil {
			t.Errorf("achievements should not be nil")
		}
	})
}

func TestLoadTasks_ConcurrentSave(t *testing.T) {
	withTempHome(t, func() {
		task := Task{
			ID:        "t1",
			Title:     "Concurrent Task",
			Priority:  Low,
			CreatedAt: time.Now(),
		}
		if err := SaveTasks([]Project{}, []Task{task}, Gamification{}); err != nil {
			t.Fatal(err)
		}

		_, tasks, _, err := LoadTasks()
		if err != nil {
			t.Fatal(err)
		}
		if len(tasks) != 1 || tasks[0].ID != "t1" {
			t.Errorf("unexpected tasks: %+v", tasks)
		}
	})
}

func TestLoadTasks_NilFieldsNormalized(t *testing.T) {
	withTempHome(t, func() {
		dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "oc-tasks")
		os.MkdirAll(dir, 0755)
		data := `{"projects":null,"tasks":null,"gamification":{"xp":0,"level":0}}`
		os.WriteFile(filepath.Join(dir, "tasks.json"), []byte(data), 0644)

		projects, tasks, gam, err := LoadTasks()
		if err != nil {
			t.Fatal(err)
		}
		if projects == nil {
			t.Errorf("nil projects should become empty slice")
		}
		if tasks == nil {
			t.Errorf("nil tasks should become empty slice")
		}
		if gam.Achievements == nil {
			t.Errorf("nil achievements should become empty slice")
		}
		if gam.Level != 1 {
			t.Errorf("Level 0 should become 1, got %d", gam.Level)
		}
	})
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Errorf("generateID returned empty string")
	}
	if id1 == id2 {
		t.Errorf("consecutive calls should produce different IDs, got %q and %q", id1, id2)
	}
}

func TestFileModTime(t *testing.T) {
	withTempHome(t, func() {
		mt := fileModTime()
		if !mt.IsZero() {
			t.Errorf("expected zero time for non-existent file")
		}

		SaveTasks([]Project{}, []Task{}, Gamification{})
		mt = fileModTime()
		if mt.IsZero() {
			t.Errorf("expected non-zero time after saving")
		}
	})
}
