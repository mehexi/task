package internal

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncateWidth(t *testing.T) {
	tests := []struct {
		s    string
		maxW int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 3, "he\xe2\x80\xa6"},
		{"hello", 5, "hell\xe2\x80\xa6"},
		{"hello", 0, ""},
		{"hello", -1, ""},
		{"", 10, ""},
		{"ab", 1, "a\xe2\x80\xa6"},
		{"a", 1, "a"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncateWidth(tt.s, tt.maxW)
			if got != tt.want {
				t.Errorf("truncateWidth(%q, %d) = %q, want %q", tt.s, tt.maxW, got, tt.want)
			}
		})
	}
}

func TestSubtaskCountsOf(t *testing.T) {
	m := Model{
		tasks: []Task{
			{ID: "parent1", Title: "Parent", ProjectID: "proj1"},
			{ID: "child1", Title: "Child 1", ParentID: "parent1", Done: true},
			{ID: "child2", Title: "Child 2", ParentID: "parent1", Done: false},
			{ID: "child3", Title: "Child 3", ParentID: "parent1", Done: true},
			{ID: "other", Title: "Other", ParentID: "otherparent"},
		},
	}

	done, total := m.subtaskCountsOf("parent1")
	if total != 3 {
		t.Errorf("expected 3 total subtasks, got %d", total)
	}
	if done != 2 {
		t.Errorf("expected 2 done subtasks, got %d", done)
	}

	done, total = m.subtaskCountsOf("nonexistent")
	if total != 0 || done != 0 {
		t.Errorf("expected 0 for nonexistent parent, got done=%d total=%d", done, total)
	}
}

func TestXpPercent(t *testing.T) {
	tests := []struct {
		name string
		gam  Gamification
		want int
	}{
		{"zero xp at level 1", Gamification{Level: 1, XP: 0}, 0},
		{"50 xp at level 1", Gamification{Level: 1, XP: 50}, 50},
		{"99 xp at level 1", Gamification{Level: 1, XP: 99}, 99},
		{"level 0", Gamification{Level: 0, XP: 0}, 0},
		{"higher level", Gamification{Level: 3, XP: 100}, 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{gamification: tt.gam}
			got := m.xpPercent()
			if got != tt.want {
				t.Errorf("xpPercent() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestXpBar(t *testing.T) {
	m := Model{gamification: Gamification{Level: 1, XP: 50}}
	bar := m.xpBar(10)
	if lipgloss.Width(bar) != 10 {
		t.Errorf("expected bar display width 10, got %d", lipgloss.Width(bar))
	}
}

func TestDisplayedTasks_Empty(t *testing.T) {
	m := Model{
		projects: []Project{},
		tasks:    []Task{},
	}
	got := m.displayedTasks()
	if len(got) != 0 {
		t.Errorf("expected 0 displayed tasks, got %d", len(got))
	}
}

func TestDisplayedTasks_FilterByProject(t *testing.T) {
	m := Model{
		projects: []Project{
			{ID: "proj1", Name: "Project A"},
			{ID: "proj2", Name: "Project B"},
		},
		tasks: []Task{
			{ID: "t1", Title: "Task A", ProjectID: "proj1"},
			{ID: "t2", Title: "Task B", ProjectID: "proj2"},
		},
		activeProj: 0,
	}

	got := m.displayedTasks()
	if len(got) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got))
	}
	if got[0].ID != "t1" {
		t.Errorf("expected task t1, got %s", got[0].ID)
	}
}

func TestDisplayedTasks_FilterByText(t *testing.T) {
	m := Model{
		projects: []Project{
			{ID: "proj1", Name: "Project"},
		},
		tasks: []Task{
			{ID: "t1", Title: "Important thing", ProjectID: "proj1"},
			{ID: "t2", Title: "Other thing", ProjectID: "proj1"},
		},
		activeProj: 0,
		filterStr:  "important",
	}

	got := m.displayedTasks()
	if len(got) != 1 {
		t.Fatalf("expected 1 filtered task, got %d", len(got))
	}
	if got[0].ID != "t1" {
		t.Errorf("expected t1, got %s", got[0].ID)
	}
}

func TestDisplayedTasks_Sorted(t *testing.T) {
	now := time.Now()
	m := Model{
		projects: []Project{
			{ID: "proj1", Name: "Project"},
		},
		tasks: []Task{
			{ID: "t1", Title: "Older done", ProjectID: "proj1", Done: true, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "t2", Title: "Newer todo", ProjectID: "proj1", Done: false, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: "t3", Title: "Older todo", ProjectID: "proj1", Done: false, CreatedAt: now.Add(-3 * time.Hour)},
		},
		activeProj: 0,
	}

	got := m.displayedTasks()
	if len(got) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(got))
	}

	if got[0].ID != "t3" {
		t.Errorf("expected first task t3 (oldest undone), got %s", got[0].ID)
	}
	if got[1].ID != "t2" {
		t.Errorf("expected second task t2, got %s", got[1].ID)
	}
	if got[2].ID != "t1" {
		t.Errorf("expected third task t1 (done), got %s", got[2].ID)
	}
}

func TestSubtasksOf(t *testing.T) {
	m := Model{
		tasks: []Task{
			{ID: "p1", ParentID: ""},
			{ID: "c1", ParentID: "p1"},
			{ID: "c2", ParentID: "p1"},
			{ID: "other", ParentID: "other"},
		},
	}

	subs := m.subtasksOf("p1")
	if len(subs) != 2 {
		t.Fatalf("expected 2 subtasks, got %d", len(subs))
	}
}

func TestFlatItems(t *testing.T) {
	m := Model{
		projects: []Project{
			{ID: "proj1", Name: "Project"},
		},
		tasks: []Task{
			{ID: "p1", Title: "Parent", ProjectID: "proj1"},
			{ID: "c1", Title: "Child", ProjectID: "proj1", ParentID: "p1"},
			{ID: "other", Title: "Other", ProjectID: "proj1"},
		},
		activeProj: 0,
	}

	items := m.flatItems()
	if len(items) != 3 {
		t.Fatalf("expected 3 flat items, got %d", len(items))
	}
	if items[0].task.ID != "p1" || items[0].level != 0 {
		t.Errorf("expected p1 at level 0, got %s at %d", items[0].task.ID, items[0].level)
	}
	if items[1].task.ID != "c1" || items[1].level != 1 {
		t.Errorf("expected c1 at level 1, got %s at %d", items[1].task.ID, items[1].level)
	}
}

func TestFlatItems_Cache(t *testing.T) {
	m := Model{
		projects: []Project{
			{ID: "proj1", Name: "Project"},
		},
		tasks: []Task{
			{ID: "t1", Title: "Task", ProjectID: "proj1"},
		},
		activeProj: 0,
	}

	items1 := m.flatItems()
	if !m.flatCacheOk {
		t.Errorf("expected flat cache to be set")
	}
	items2 := m.flatItems()
	if len(items1) != len(items2) {
		t.Errorf("cache returned different results")
	}
}

func TestInvaldateFlatCache(t *testing.T) {
	m := Model{}
	m.invalidateFlatCache()
	if m.flatCacheOk {
		t.Errorf("cache should be invalid after invalidation")
	}
}

func TestToggleAllChildren(t *testing.T) {
	m := &Model{
		tasks: []Task{
			{ID: "p1", Done: false},
			{ID: "c1", ParentID: "p1", Done: false},
			{ID: "c2", ParentID: "p1", Done: true},
			{ID: "other", Done: false},
		},
	}

	m.toggleAllChildren("p1", true)

	for _, tsk := range m.tasks {
		if tsk.ParentID == "p1" && !tsk.Done {
			t.Errorf("child %s should be done after toggle", tsk.ID)
		}
	}

	if m.tasks[0].Done {
		t.Errorf("parent should not be toggled by toggleAllChildren")
	}
	if m.tasks[3].Done {
		t.Errorf("unrelated task should not be toggled")
	}
}

func TestAwardXP_BasicTask(t *testing.T) {
	m := &Model{
		gamification: Gamification{
			Level:    1,
			XP:       0,
			DailyLog: make(map[string]int),
		},
	}

	task := Task{
		ID:       "t1",
		Title:    "Simple task",
		Priority: Low,
	}

	m.awardXP(task)

	if m.gamification.XP != 10 {
		t.Errorf("expected 10 XP for low priority task, got %d", m.gamification.XP)
	}
	if m.gamification.TotalCompleted != 1 {
		t.Errorf("expected TotalCompleted=1, got %d", m.gamification.TotalCompleted)
	}
	if m.gamification.Level != 1 {
		t.Errorf("level should remain 1 at 10 XP, got %d", m.gamification.Level)
	}
}

func TestAwardXP_PriorityBonuses(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		expected int
	}{
		{"low", Low, 10},
		{"medium", Medium, 15},
		{"high", High, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				gamification: Gamification{
					Level:    1,
					XP:       0,
					DailyLog: make(map[string]int),
				},
			}
			task := Task{ID: "t1", Title: "Task", Priority: tt.priority}
			m.awardXP(task)
			if m.gamification.XP != tt.expected {
				t.Errorf("expected %d XP for %s priority, got %d", tt.expected, tt.name, m.gamification.XP)
			}
		})
	}
}

func TestAwardXP_HighCompletedCount(t *testing.T) {
	m := &Model{
		gamification: Gamification{
			Level:    1,
			XP:       0,
			DailyLog: make(map[string]int),
		},
	}

	m.awardXP(Task{ID: "t1", Title: "High", Priority: High})
	if m.gamification.HighCompleted != 1 {
		t.Errorf("expected HighCompleted=1, got %d", m.gamification.HighCompleted)
	}

	m.awardXP(Task{ID: "t2", Title: "Low", Priority: Low})
	if m.gamification.HighCompleted != 1 {
		t.Errorf("HighCompleted should remain 1, got %d", m.gamification.HighCompleted)
	}
}

func TestCheckAchievements_FirstStep(t *testing.T) {
	m := &Model{
		gamification: Gamification{
			TotalCompleted: 1,
			Achievements:   []string{},
		},
	}

	ach := m.checkAchievements(Task{})
	if ach != "First Step" {
		t.Errorf("expected 'First Step' achievement, got %q", ach)
	}
	if len(m.gamification.Achievements) != 1 {
		t.Errorf("expected 1 achievement, got %d", len(m.gamification.Achievements))
	}
}

func TestCheckAchievements_TenTasks(t *testing.T) {
	m := &Model{
		gamification: Gamification{
			TotalCompleted: 10,
			Achievements:   []string{"first_step"},
		},
	}

	ach := m.checkAchievements(Task{})
	if ach != "Getting Started" {
		t.Errorf("expected 'Getting Started', got %q", ach)
	}
}

func TestCheckAchievements_NoDoubleAward(t *testing.T) {
	m := &Model{
		projects: []Project{{ID: "proj1", Name: "Proj"}},
		tasks:    []Task{{ID: "t1", Title: "Task", ProjectID: "proj1", Done: false}},
		gamification: Gamification{
			TotalCompleted: 1,
			Achievements:   []string{"first_step"},
			DailyLog:       make(map[string]int),
		},
	}

	ach := m.checkAchievements(Task{})
	if ach == "First Step" {
		t.Errorf("should not re-award 'First Step'")
	}
}

func TestCheckAchievements_FiveProjects(t *testing.T) {
	m := &Model{
		projects: []Project{
			{ID: "1", Name: "A"},
			{ID: "2", Name: "B"},
			{ID: "3", Name: "C"},
			{ID: "4", Name: "D"},
			{ID: "5", Name: "E"},
		},
		gamification: Gamification{
			Achievements: []string{},
		},
	}

	ach := m.checkAchievements(Task{})
	if ach != "Organized" {
		t.Errorf("expected 'Organized', got %q", ach)
	}
}

func TestUpdateStreak_FirstCompletion(t *testing.T) {
	m := &Model{
		gamification: Gamification{
			DailyLog: make(map[string]int),
		},
	}

	m.updateStreak()
	if m.gamification.Streak != 1 {
		t.Errorf("expected streak 1 on first completion, got %d", m.gamification.Streak)
	}
	if m.gamification.LongestStreak != 1 {
		t.Errorf("expected longest streak 1, got %d", m.gamification.LongestStreak)
	}
}

func TestNew(t *testing.T) {
	m := New()
	if m.focus != focusTasks {
		t.Errorf("expected focus focusTasks, got %d", m.focus)
	}
	if m.mode != modeNormal {
		t.Errorf("expected modeNormal, got %d", m.mode)
	}
	if m.keys.Up.Keys() == nil {
		t.Errorf("expected keys to be initialized")
	}
}

func TestSourceInfo_OriginalKind(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"DONE", "TODO"},
		{"TODO", "TODO"},
		{"FIXME", "FIXME"},
		{"HACK", "HACK"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			s := &SourceInfo{Kind: tt.kind}
			got := s.OriginalKind()
			if got != tt.want {
				t.Errorf("OriginalKind() = %q, want %q", got, tt.want)
			}
		})
	}
}
