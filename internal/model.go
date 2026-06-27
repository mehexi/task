package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusProjects focus = iota
	focusTasks
	focusDetail
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeNewTask
	modeNewProject
	modeFilter
	modeDueDate
	modeEditNotes
	modeConfirmDeleteTask
	modeConfirmDeleteProj
)

type tasksLoadedMsg struct {
	projects []Project
	tasks    []Task
}

type scanResultsMsg struct {
	items   []CommentItem
	rootDir string
	err     error
}

type scanDoneMsg struct{}

type errMsg struct{ err error }

type tickMsg time.Time

type flatItem struct {
	task  Task
	level int
}

type Model struct {
	projects []Project
	tasks    []Task

	activeProj int
	activeTask int
	taskScroll int
	focus      focus
	mode       inputMode

	width  int
	height int
	ready  bool

	newTaskInput    textinput.Model
	newProjectInput textinput.Model
	filterInput     textinput.Model
	dueDateInput    textinput.Model
	notesTextarea   textarea.Model

	viewport  viewport.Model
	vpContent string
	filterStr string
	lastMod   time.Time

	keys          KeyMap
	codeRoot      string
	scanning      bool
	subtaskParent string

	flatCache     []flatItem
	flatCacheOk   bool
}

func New() Model {
	return Model{
		keys:       Keys,
		activeProj: 0,
		activeTask: 0,
		focus:      focusTasks,
		mode:       modeNormal,
		viewport:   viewport.New(80, 20),
	}
}

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func fileModTime() time.Time {
	path := dataFile()
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			projects, tasks, err := LoadTasks()
			if err != nil {
				return errMsg{err}
			}
			return tasksLoadedMsg{projects, tasks}
		},
		tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

func codeDetectCmd() tea.Cmd {
	return func() tea.Msg {
		root, err := os.Getwd()
		if err != nil {
			return scanResultsMsg{err: err}
		}
		entries, err := os.ReadDir(root)
		if err == nil && len(entries) > 5000 {
			return scanResultsMsg{rootDir: root}
		}
		items, err := FindComments(root)
		return scanResultsMsg{items: items, rootDir: root, err: err}
	}
}

func (m *Model) updateLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	paneH := m.height - 4
	if paneH < 3 {
		paneH = 3
	}
	projW := max(10, int(float64(m.width-6)*0.2))
	taskW := max(15, int(float64(m.width-6)*0.4))
	detailW := max(15, m.width-6-projW-taskW)

	vpW := detailW - 4
	vpH := paneH - 4
	if vpW < 5 {
		vpW = 5
	}
	if vpH < 1 {
		vpH = 1
	}
	m.viewport.Width = vpW
	m.viewport.Height = vpH
	m.updateViewportContent()
}

func (m *Model) updateViewportContent() {
	items := m.flatItems()
	if len(items) == 0 || m.activeTask >= len(items) {
		m.viewport.SetContent(TitleS.Render("DETAIL"))
		return
	}
	t := items[m.activeTask].task
	for i := range m.tasks {
		if m.tasks[i].ID == t.ID {
			t = m.tasks[i]
			break
		}
	}

	var b strings.Builder
	b.WriteString(TitleS.Render("DETAIL"))
	b.WriteString("\n\n")
	b.WriteString(m.renderField("Title", t.Title))
	b.WriteString("\n")
	projName := ""
	for _, p := range m.projects {
		if p.ID == t.ProjectID {
			projName = p.Name
			break
		}
	}
	b.WriteString(m.renderField("Project", projName))
	b.WriteString("\n")
	b.WriteString(m.renderField("Priority", string(t.Priority)))
	b.WriteString("\n")
	status := "○ Todo"
	if t.Done {
		status = "● Done"
	}
	b.WriteString(m.renderField("Status", status))
	b.WriteString("\n")
	if t.DueDate != "" {
		b.WriteString(m.renderField("Due", t.DueDate))
		b.WriteString("\n")
	}
	if done, total := m.subtaskCountsOf(t.ID); total > 0 {
		b.WriteString(m.renderField("Subtasks", fmt.Sprintf("%d/%d done", done, total)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(LabelS.Render("Notes"))
	b.WriteString("\n")
	if t.Notes != "" {
		b.WriteString(t.Notes)
	}
	if t.ParentID != "" {
		for _, p := range m.tasks {
			if p.ID == t.ParentID {
				b.WriteString("\n\n")
				b.WriteString(DimS.Render("Parent task: " + p.Title))
				break
			}
		}
	}

	m.viewport.SetContent(b.String())
}

func (m Model) renderField(label, value string) string {
	return LabelS.Render(fmt.Sprintf("%-10s", label+":")) + "  " + ValueS.Render(value)
}

func (m Model) displayedTasks() []Task {
	if len(m.projects) == 0 || m.activeProj >= len(m.projects) {
		return nil
	}
	projID := m.projects[m.activeProj].ID
	var result []Task
	for _, t := range m.tasks {
		if t.ProjectID != projID {
			continue
		}
		if m.filterStr != "" && !strings.Contains(strings.ToLower(t.Title), strings.ToLower(m.filterStr)) {
			continue
		}
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Done != result[j].Done {
			return !result[i].Done
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func (m Model) subtasksOf(parentID string) []Task {
	var result []Task
	for _, t := range m.tasks {
		if t.ParentID == parentID {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Done != result[j].Done {
			return !result[i].Done
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func (m Model) subtaskCountsOf(parentID string) (done, total int) {
	for _, t := range m.tasks {
		if t.ParentID == parentID {
			total++
			if t.Done {
				done++
			}
		}
	}
	return
}

func (m *Model) invalidateFlatCache() {
	m.flatCacheOk = false
}

func (m *Model) flatItems() []flatItem {
	if m.flatCacheOk {
		return m.flatCache
	}
	tasks := m.displayedTasks()
	items := make([]flatItem, 0, len(tasks))
	for _, t := range tasks {
		if t.ParentID != "" {
			continue
		}
		items = append(items, flatItem{t, 0})
		children := m.subtasksOf(t.ID)
		for _, c := range children {
			items = append(items, flatItem{c, 1})
		}
	}
	m.flatCache = items
	m.flatCacheOk = true
	return items
}

func (m *Model) save() {
	if err := SaveTasks(m.projects, m.tasks); err != nil {
		return
	}
	if fi, err := os.Stat(dataFile()); err == nil {
		m.lastMod = fi.ModTime()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		m.updateLayout()
		if m.mode == modeEditNotes {
			m.notesTextarea.SetWidth(m.viewport.Width)
			m.notesTextarea.SetHeight(m.viewport.Height)
		}

	case tasksLoadedMsg:
		m.projects = msg.projects
		m.tasks = msg.tasks
		m.lastMod = fileModTime()
		m.invalidateFlatCache()
		m.updateLayout()
		return m, codeDetectCmd()

	case scanResultsMsg:
		if msg.err != nil {
			return m, nil
		}
		m.codeRoot = msg.rootDir
		if len(msg.items) == 0 {
			return m, nil
		}
		projName := filepath.Base(msg.rootDir)
		if projName == "" || projName == "." || projName == "/" {
			projName = "Code"
		}
		var projID string
		for _, p := range m.projects {
			if p.CodeDir != "" && p.CodeDir == msg.rootDir {
				projID = p.ID
				break
			}
		}
		if projID == "" {
			for _, p := range m.projects {
				if strings.EqualFold(p.Name, projName) {
					projID = p.ID
					break
				}
			}
		}
		if projID == "" {
			p := Project{
				ID:      generateID(),
				Name:    projName,
				Color:   "#7aa2f7",
				CodeDir: msg.rootDir,
			}
			m.projects = append(m.projects, p)
			projID = p.ID
		} else {
			for i := range m.projects {
				if m.projects[i].ID == projID && m.projects[i].CodeDir == "" {
					m.projects[i].CodeDir = msg.rootDir
				}
			}
		}

		sourceKey := func(item CommentItem) string {
			return item.File + ":" + fmt.Sprint(item.Line)
		}
		existing := make(map[string]bool)
		for _, t := range m.tasks {
			if t.Source != nil {
				existing[t.Source.File+":"+fmt.Sprint(t.Source.Line)] = true
			}
		}

		for _, item := range msg.items {
			if existing[sourceKey(item)] {
				continue
			}
			kindPrefix := "[" + item.Kind + "]"
			title := kindPrefix + " " + item.Text
			notes := item.File + ":" + fmt.Sprint(item.Line)

			t := Task{
				ID:        generateID(),
				Title:     title,
				Done:      false,
				Priority:  Low,
				ProjectID: projID,
				Notes:     notes,
				CreatedAt: time.Now(),
				Source: &SourceInfo{
					File:     filepath.Join(msg.rootDir, item.File),
					Line:     item.Line,
					Original: item.Original,
					Kind:     item.Kind,
				},
			}
			m.tasks = append(m.tasks, t)
		}

		codeProjIdx := -1
		for i, p := range m.projects {
			if p.ID == projID {
				codeProjIdx = i
				break
			}
		}
		if codeProjIdx >= 0 {
			cur := m.projects[m.activeProj]
			if cur.ID == projID || cur.CodeDir == "" {
				m.activeProj = codeProjIdx
			}
		}
		m.scanning = false
		m.invalidateFlatCache()
		m.updateViewportContent()
		m.save()
		return m, nil

	case scanDoneMsg:
		m.scanning = false
		return m, nil

	case errMsg:
		if m.projects == nil {
			m.projects = []Project{}
		}
		if m.tasks == nil {
			m.tasks = []Task{}
		}

	case tickMsg:
		if m.mode == modeNormal {
			path := dataFile()
			fi, err := os.Stat(path)
			if err == nil && !fi.ModTime().Equal(m.lastMod) {
				m.lastMod = fi.ModTime()
				if projects, tasks, err := LoadTasks(); err == nil {
					oldProjID := ""
					if m.activeProj < len(m.projects) {
						oldProjID = m.projects[m.activeProj].ID
					}
					oldTaskID := ""
					fl := m.flatItems()
					if m.activeTask < len(fl) {
						oldTaskID = fl[m.activeTask].task.ID
					}
					m.projects = projects
					m.tasks = tasks
					m.invalidateFlatCache()
					m.activeProj = 0
					for i, p := range m.projects {
						if p.ID == oldProjID {
							m.activeProj = i
							break
						}
					}
					if m.activeProj >= len(m.projects) {
						m.activeProj = max(0, len(m.projects)-1)
					}
					nfl := m.flatItems()
					m.activeTask = 0
					for i, it := range nfl {
						if it.task.ID == oldTaskID {
							m.activeTask = i
							break
						}
					}
					if m.activeTask >= len(nfl) {
						m.activeTask = max(0, len(nfl)-1)
					}
					m.taskScroll = 0
					m.updateViewportContent()
				}
			}
		}
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeNewTask:
		return m.handleNewTaskInput(msg)
	case modeNewProject:
		return m.handleNewProjectInput(msg)
	case modeFilter:
		return m.handleFilterInput(msg)
	case modeDueDate:
		return m.handleDueDateInput(msg)
	case modeEditNotes:
		return m.handleEditNotesInput(msg)
	case modeConfirmDeleteTask:
		return m.handleConfirmDeleteTask(msg)
	case modeConfirmDeleteProj:
		return m.handleConfirmDeleteProj(msg)
	default:
		return m.handleNormalKeys(msg)
	}
}

func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.save()
		return m, tea.Quit
	case key.Matches(msg, m.keys.Tab):
		m.focus = (m.focus + 1) % 3
		return m, nil
	case key.Matches(msg, m.keys.Focus1):
		m.focus = focusProjects
		return m, nil
	case key.Matches(msg, m.keys.Focus2):
		m.focus = focusTasks
		return m, nil
	case key.Matches(msg, m.keys.Focus3):
		m.focus = focusDetail
		return m, nil
	}

	switch m.focus {
	case focusProjects:
		return m.handleProjectsKeys(msg)
	case focusTasks:
		return m.handleTasksKeys(msg)
	case focusDetail:
		return m.handleDetailKeys(msg)
	}
	return m, nil
}

func (m Model) handleProjectsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if len(m.projects) > 0 {
			m.activeProj = (m.activeProj - 1 + len(m.projects)) % len(m.projects)
			m.activeTask = 0
			m.taskScroll = 0
			m.updateViewportContent()
		}
	case key.Matches(msg, m.keys.Down):
		if len(m.projects) > 0 {
			m.activeProj = (m.activeProj + 1) % len(m.projects)
			m.activeTask = 0
			m.taskScroll = 0
			m.updateViewportContent()
		}
	case key.Matches(msg, m.keys.Enter):
		if len(m.projects) > 0 {
			m.focus = focusTasks
			m.activeTask = 0
			m.updateViewportContent()
		}
	case key.Matches(msg, m.keys.New):
		m.newProjectInput = textinput.New()
		m.newProjectInput.Placeholder = "Project name..."
		m.newProjectInput.Focus()
		m.mode = modeNewProject
	case key.Matches(msg, m.keys.Delete):
		if len(m.projects) > 0 {
			m.mode = modeConfirmDeleteProj
		}
	}
	return m, nil
}

func (m Model) handleTasksKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.flatItems()

	switch {
	case key.Matches(msg, m.keys.Up):
		if len(items) > 0 {
			m.activeTask = (m.activeTask - 1 + len(items)) % len(items)
			m.clampTaskScroll()
			m.updateViewportContent()
		}
	case key.Matches(msg, m.keys.Down):
		if len(items) > 0 {
			m.activeTask = (m.activeTask + 1) % len(items)
			m.clampTaskScroll()
			m.updateViewportContent()
		}
	case key.Matches(msg, m.keys.Space):
		if len(items) > 0 {
			t := items[m.activeTask].task
			for i := range m.tasks {
				if m.tasks[i].ID == t.ID {
					m.tasks[i].Done = !m.tasks[i].Done
					if m.tasks[i].Source != nil {
						updateSourceComment(m.tasks[i].Source, m.tasks[i].Done)
					}
					if m.tasks[i].ParentID == "" {
						m.toggleAllChildren(m.tasks[i].ID, m.tasks[i].Done)
					}
					break
				}
			}
			m.invalidateFlatCache()
			m.updateViewportContent()
			m.save()
		}
	case key.Matches(msg, m.keys.New):
		if len(m.projects) == 0 {
			m.focus = focusProjects
			m.newProjectInput = textinput.New()
			m.newProjectInput.Placeholder = "Project name..."
			m.newProjectInput.Focus()
			m.mode = modeNewProject
			return m, nil
		}
		m.subtaskParent = ""
		m.newTaskInput = textinput.New()
		m.newTaskInput.Placeholder = "Task title..."
		m.newTaskInput.Focus()
		m.mode = modeNewTask
	case key.Matches(msg, m.keys.Subtask):
		if len(items) > 0 {
			parent := items[m.activeTask].task
			m.subtaskParent = parent.ID
			m.newTaskInput = textinput.New()
			m.newTaskInput.Placeholder = "Subtask title..."
			m.newTaskInput.Focus()
			m.mode = modeNewTask
		}
	case key.Matches(msg, m.keys.Delete):
		if len(items) > 0 {
			m.mode = modeConfirmDeleteTask
		}
	case key.Matches(msg, m.keys.CyclePriority):
		if len(items) > 0 {
			t := items[m.activeTask].task
			for i := range m.tasks {
				if m.tasks[i].ID == t.ID {
					switch m.tasks[i].Priority {
					case Low:
						m.tasks[i].Priority = Medium
					case Medium:
						m.tasks[i].Priority = High
					case High:
						m.tasks[i].Priority = Low
					}
					break
				}
			}
			m.invalidateFlatCache()
			m.updateViewportContent()
			m.save()
		}
	case key.Matches(msg, m.keys.Enter):
		if len(items) > 0 {
			m.focus = focusDetail
		}
	case key.Matches(msg, m.keys.Filter):
		m.filterInput = textinput.New()
		m.filterInput.Placeholder = "Filter tasks..."
		m.filterInput.SetValue(m.filterStr)
		m.filterInput.Focus()
		m.mode = modeFilter
	case key.Matches(msg, m.keys.Rescan):
		if !m.scanning {
			m.scanning = true
			return m, rescanCmd(m.codeRoot)
		}
	}
	return m, nil
}

func (m *Model) toggleAllChildren(parentID string, done bool) {
	for i := range m.tasks {
		if m.tasks[i].ParentID == parentID {
			m.tasks[i].Done = done
			if m.tasks[i].Source != nil {
				updateSourceComment(m.tasks[i].Source, done)
			}
		}
	}
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.viewport.LineUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.viewport.LineDown(1)
		return m, nil
	case key.Matches(msg, m.keys.Escape):
		m.focus = focusTasks
	case key.Matches(msg, m.keys.EditNotes):
		items := m.flatItems()
		if len(items) > 0 && m.activeTask < len(items) {
			t := items[m.activeTask].task
			for i := range m.tasks {
				if m.tasks[i].ID == t.ID {
					m.notesTextarea = textarea.New()
					m.notesTextarea.SetValue(m.tasks[i].Notes)
					m.notesTextarea.SetWidth(m.viewport.Width)
					m.notesTextarea.SetHeight(m.viewport.Height)
					m.notesTextarea.Focus()
					m.mode = modeEditNotes
					break
				}
			}
		}
	case key.Matches(msg, m.keys.SetDueDate):
		items := m.flatItems()
		if len(items) > 0 && m.activeTask < len(items) {
			t := items[m.activeTask].task
			m.dueDateInput = textinput.New()
			m.dueDateInput.Placeholder = "YYYY-MM-DD"
			for i := range m.tasks {
				if m.tasks[i].ID == t.ID {
					if m.tasks[i].DueDate != "" {
						m.dueDateInput.SetValue(m.tasks[i].DueDate)
					}
					break
				}
			}
			m.dueDateInput.Focus()
			m.mode = modeDueDate
		}
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleNewTaskInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		title := strings.TrimSpace(m.newTaskInput.Value())
		if title != "" && len(m.projects) > 0 {
			t := Task{
				ID:        generateID(),
				Title:     title,
				Done:      false,
				Priority:  Low,
				ProjectID: m.projects[m.activeProj].ID,
				ParentID:  m.subtaskParent,
				DueDate:   "",
				Notes:     "",
				CreatedAt: time.Now(),
			}
			m.tasks = append(m.tasks, t)
			m.invalidateFlatCache()
			items := m.flatItems()
			m.activeTask = max(0, len(items)-1)
			m.clampTaskScroll()
			m.updateViewportContent()
			m.save()
		}
		m.mode = modeNormal
		m.subtaskParent = ""
		m.newTaskInput = textinput.Model{}
	case "esc":
		m.mode = modeNormal
		m.newTaskInput = textinput.Model{}
	default:
		var cmd tea.Cmd
		m.newTaskInput, cmd = m.newTaskInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleNewProjectInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.newProjectInput.Value())
		if name != "" {
			p := Project{
				ID:    generateID(),
				Name:  name,
				Color: "#7aa2f7",
			}
			m.projects = append(m.projects, p)
			m.invalidateFlatCache()
			m.activeProj = len(m.projects) - 1
			m.activeTask = 0
			m.updateViewportContent()
			m.save()
		}
		m.mode = modeNormal
		m.newProjectInput = textinput.Model{}
	case "esc":
		m.mode = modeNormal
		m.newProjectInput = textinput.Model{}
	default:
		var cmd tea.Cmd
		m.newProjectInput, cmd = m.newProjectInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filterStr = m.filterInput.Value()
		m.mode = modeNormal
		m.filterInput = textinput.Model{}
		m.invalidateFlatCache()
		if len(m.displayedTasks()) > 0 {
			m.activeTask = 0
			m.taskScroll = 0
		}
		m.updateViewportContent()
	case "esc":
		m.filterStr = ""
		m.mode = modeNormal
		m.filterInput = textinput.Model{}
		m.invalidateFlatCache()
		m.taskScroll = 0
		m.updateViewportContent()
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.filterStr = m.filterInput.Value()
		m.invalidateFlatCache()
		m.updateViewportContent()
		return m, cmd
	}
	return m, nil
}

func (m Model) handleDueDateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		date := strings.TrimSpace(m.dueDateInput.Value())
		items := m.flatItems()
		if len(items) > 0 && m.activeTask < len(items) {
			t := items[m.activeTask].task
			for i := range m.tasks {
				if m.tasks[i].ID == t.ID {
					m.tasks[i].DueDate = date
					break
				}
			}
			m.updateViewportContent()
			m.save()
		}
		m.mode = modeNormal
		m.dueDateInput = textinput.Model{}
	case "esc":
		m.mode = modeNormal
		m.dueDateInput = textinput.Model{}
	default:
		var cmd tea.Cmd
		m.dueDateInput, cmd = m.dueDateInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleConfirmDeleteTask(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		items := m.flatItems()
		if len(items) > 0 && m.activeTask < len(items) {
			t := items[m.activeTask].task
			var updated []Task
			for _, task := range m.tasks {
				if task.ID != t.ID && task.ParentID != t.ID {
					updated = append(updated, task)
				}
			}
			m.tasks = updated
			m.invalidateFlatCache()
			ni := m.flatItems()
			if m.activeTask >= len(ni) {
				m.activeTask = max(0, len(ni)-1)
			}
			m.updateViewportContent()
			m.save()
		}
		m.mode = modeNormal
	default:
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) handleConfirmDeleteProj(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.activeProj < len(m.projects) {
			projID := m.projects[m.activeProj].ID
			var updatedProjs []Project
			for _, p := range m.projects {
				if p.ID != projID {
					updatedProjs = append(updatedProjs, p)
				}
			}
			m.projects = updatedProjs

			var updatedTasks []Task
			for _, t := range m.tasks {
				if t.ProjectID != projID {
					updatedTasks = append(updatedTasks, t)
				}
			}
			m.tasks = updatedTasks
			m.invalidateFlatCache()

			if m.activeProj >= len(m.projects) {
				m.activeProj = max(0, len(m.projects)-1)
			}
			m.activeTask = 0
			m.updateViewportContent()
			m.save()
		}
		m.mode = modeNormal
	default:
		m.mode = modeNormal
	}
	return m, nil
}

func (m Model) handleEditNotesInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		items := m.flatItems()
		if len(items) > 0 && m.activeTask < len(items) {
			t := items[m.activeTask].task
			for i := range m.tasks {
				if m.tasks[i].ID == t.ID {
					m.tasks[i].Notes = m.notesTextarea.Value()
					break
				}
			}
		}
		m.mode = modeNormal
		m.notesTextarea.Blur()
		m.updateViewportContent()
		m.save()
	default:
		var cmd tea.Cmd
		m.notesTextarea, cmd = m.notesTextarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.headerView(),
		m.bodyView(),
		m.statusView(),
	)
}

func (m Model) headerView() string {
	left := AccentS.Render(" oc-tasks ")
	right := DimS.Render("[i] inbox  [t] today  [p] project")
	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}
	line := left + strings.Repeat(" ", padding) + right
	return HeaderS.Width(m.width).Render(line)
}

func (m Model) statusView() string {
	var text string
	switch m.mode {
	case modeNormal:
		switch m.focus {
		case focusProjects:
			text = "j/k navigate • enter select • n new project • d delete"
		case focusTasks:
			text = "j/k • space toggle • n new • a subtask • d delete • p priority • / filter • R rescan"
		case focusDetail:
			text = "e edit notes • D set due date • esc back"
		}
	case modeNewTask:
		text = "New task: " + m.newTaskInput.View()
	case modeNewProject:
		text = "New project: " + m.newProjectInput.View()
	case modeFilter:
		text = "Filter: " + m.filterInput.View()
	case modeDueDate:
		text = "Due date (YYYY-MM-DD): " + m.dueDateInput.View()
	case modeConfirmDeleteTask:
		text = "Delete this task? (y/n)"
	case modeConfirmDeleteProj:
		text = "Delete this project and all tasks? (y/n)"
	}
	return StatusS.Width(m.width).Render(text)
}

func (m Model) bodyView() string {
	paneH := m.height - 4
	if paneH < 3 {
		paneH = 3
	}
	projW := max(10, int(float64(m.width-6)*0.2))
	taskW := max(15, int(float64(m.width-6)*0.4))
	detailW := max(15, m.width-6-projW-taskW)

	proj := m.projectsPane(projW, paneH)
	task := m.tasksPane(taskW, paneH)
	detail := m.detailPane(detailW, paneH)

	return lipgloss.JoinHorizontal(lipgloss.Top, proj, task, detail)
}

func (m Model) projectsPane(w, h int) string {
	var b strings.Builder
	for i, p := range m.projects {
		prefix := "  "
		if i == m.activeProj {
			prefix = "▶ "
		}
		line := prefix + p.Name
		if i == m.activeProj {
			b.WriteString(ProjActiveS.Render(ActiveProjBg.Render(line)))
		} else {
			b.WriteString(ProjInactiveS.Render(line))
		}
		b.WriteString("\n")
	}
	if len(m.projects) == 0 {
		b.WriteString(DimS.Render("No projects"))
		b.WriteString("\n")
		b.WriteString(DimS.Render("Press 'n' to create"))
	}

	style := InactiveBorder.Width(w).Height(h)
	if m.focus == focusProjects && m.mode == modeNormal {
		style = ActiveBorder.Width(w).Height(h)
	}
	return style.Render(TitleS.Render("PROJECTS") + "\n" + b.String())
}

func (m *Model) clampTaskScroll() {
	paneH := max(1, m.height-8) // body height minus borders/title overhead
	if paneH < 1 {
		m.taskScroll = 0
		return
	}
	if m.activeTask < m.taskScroll {
		m.taskScroll = m.activeTask
	}
	if m.activeTask >= m.taskScroll+paneH {
		m.taskScroll = m.activeTask - paneH + 1
	}
}

func (m Model) tasksPane(w, h int) string {
	contentW := w - 4
	maxLines := max(1, h-4)
	items := m.flatItems()

	var b strings.Builder
	shown := 0
	for i := m.taskScroll; i < len(items) && shown < maxLines; i++ {
		shown++
		it := items[i]
		line := m.taskLine(it.task, contentW, it.level)
		if i == m.activeTask {
			line = ActiveProjBg.Render(line)
		}
		b.WriteString(line)
		if shown < maxLines {
			b.WriteString("\n")
		}
	}
	if len(items) == 0 {
		if len(m.projects) == 0 {
			b.WriteString(DimS.Render("No projects"))
		} else if m.filterStr != "" {
			b.WriteString(DimS.Render("No matches"))
		} else {
			b.WriteString(DimS.Render("No tasks"))
		}
	} else if shown == 0 && len(items) > 0 {
		b.WriteString(DimS.Render("(scroll up)"))
	}

	style := InactiveBorder.Width(w).Height(h)
	if m.focus == focusTasks && m.mode == modeNormal {
		style = ActiveBorder.Width(w).Height(h)
	}
	return style.Render(TitleS.Render("TASKS") + "\n" + b.String())
}

func (m Model) taskLine(t Task, w int, level int) string {
	indicator := "○"
	if t.Done {
		indicator = "●"
	}
	prefix := ""
	for i := 0; i < level; i++ {
		prefix += "  "
	}
	if level > 0 {
		prefix += "└ "
	}
	titlePart := prefix + indicator + " " + t.Title

	var badge string
	switch t.Priority {
	case High:
		badge = PHighS.Render(" HIGH ")
	case Medium:
		badge = PMedS.Render(" MID ")
	default:
		badge = PLowS.Render(" LOW ")
	}
	badgeW := lipgloss.Width(badge)

	// Truncate title so the line fits within w
	maxTitleW := max(1, w-badgeW-1)
	if lipgloss.Width(titlePart) > maxTitleW {
		titlePart = truncateWidth(titlePart, maxTitleW)
	}

	titleW := lipgloss.Width(titlePart)
	padding := w - titleW - badgeW
	if padding < 1 {
		padding = 1
	}
	spaces := strings.Repeat(" ", padding)

	if t.Done {
		return DoneS.Render(titlePart) + spaces + badge
	}
	return titlePart + spaces + badge
}

func truncateWidth(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	var b strings.Builder
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > maxW-1 && i > 0 {
			b.WriteRune('…')
			break
		}
		b.WriteRune(r)
		w += rw
		if w >= maxW {
			if i < len(runes)-1 {
				b.WriteRune('…')
			}
			break
		}
	}
	return b.String()
}

func (m Model) detailPane(w, h int) string {
	style := InactiveBorder.Width(w).Height(h)
	if m.focus == focusDetail && m.mode == modeNormal {
		style = ActiveBorder.Width(w).Height(h)
	}

	if m.mode == modeEditNotes {
		return style.Render(m.notesTextarea.View())
	}

	return style.Render(m.viewport.View())
}

func updateSourceComment(s *SourceInfo, done bool) {
	if s == nil {
		return
	}
	data, err := os.ReadFile(s.File)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	if s.Line < 1 || s.Line > len(lines) {
		return
	}
	line := lines[s.Line-1]
	if !strings.Contains(line, s.Original) {
		return
	}
	if done {
		lines[s.Line-1] = strings.Replace(line, s.Kind, "DONE", 1)
		s.Kind = "DONE"
	} else {
		lines[s.Line-1] = strings.Replace(line, "DONE", s.OriginalKind(), 1)
		s.Kind = s.OriginalKind()
	}
	os.WriteFile(s.File, []byte(strings.Join(lines, "\n")), 0644)
}

func (s *SourceInfo) OriginalKind() string {
	switch s.Kind {
	case "DONE":
		return "TODO"
	default:
		return s.Kind
	}
}

func rescanCmd(rootDir string) tea.Cmd {
	return func() tea.Msg {
		if rootDir == "" {
			var err error
			rootDir, err = os.Getwd()
			if err != nil {
				return scanResultsMsg{err: err}
			}
		}
		items, err := FindComments(rootDir)
		return scanResultsMsg{items: items, rootDir: rootDir, err: err}
	}
}
