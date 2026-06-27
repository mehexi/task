package internal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var codeExts = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".rs": true, ".rb": true, ".java": true, ".c": true, ".h": true, ".cpp": true,
	".hpp": true, ".cs": true, ".swift": true, ".kt": true, ".scala": true,
	".sh": true, ".bash": true, ".zsh": true, ".lua": true, ".php": true, ".pl": true,
	".pm": true, ".css": true, ".scss": true, ".less": true, ".html": true, ".xml": true,
	".yaml": true, ".yml": true, ".toml": true, ".json": true, ".md": true, ".sql": true,
}

var commentPattern = regexp.MustCompile(
	`(?i)(TODO|FIXME|HACK|NOTE|XXX)\s*[:)]?\s*(.*)`,
)

type CommentItem struct {
	File     string
	Line     int
	Text     string
	Kind     string
	Original string
}

type gitignoreMatcher struct {
	patterns []gitignorePattern
}

type gitignorePattern struct {
	raw     string
	negate  bool
	dirOnly bool
	anchored bool
}

func parseGitignoreLine(line string) (gitignorePattern, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return gitignorePattern{}, false
	}
	negate := false
	if strings.HasPrefix(line, "!") {
		negate = true
		line = line[1:]
	}
	dirOnly := strings.HasSuffix(line, "/")
	if dirOnly {
		line = strings.TrimSuffix(line, "/")
	}
	anchored := strings.HasPrefix(line, "/")
	if anchored {
		line = line[1:]
	}
	return gitignorePattern{raw: line, negate: negate, dirOnly: dirOnly, anchored: anchored}, true
}

func (g gitignorePattern) match(name string, isDir bool) bool {
	if g.dirOnly && !isDir {
		return false
	}
	return matchGlob(g.raw, name)
}

func matchGlob(pattern, name string) bool {
	parts := strings.Split(pattern, "/")
	if len(parts) == 1 && !strings.Contains(pattern, "**") {
		matched, _ := filepath.Match(pattern, name)
		return matched
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}

const maxScanFiles = 20000

type scanner struct {
	root      string
	ignores   map[string]*gitignoreMatcher
	fileCount int
	maxFiles  int
}

func newScanner(root string) *scanner {
	return &scanner{
		root:      root,
		ignores:   make(map[string]*gitignoreMatcher),
		maxFiles:  maxScanFiles,
	}
}

func (s *scanner) loadGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	m := &gitignoreMatcher{}
	for _, line := range strings.Split(string(data), "\n") {
		p, ok := parseGitignoreLine(line)
		if ok {
			m.patterns = append(m.patterns, p)
		}
	}
	if len(m.patterns) > 0 {
		s.ignores[dir] = m
	}
}

func (s *scanner) isIgnored(relPath string, isDir bool) bool {
	parts := strings.Split(relPath, string(filepath.Separator))
	for i := range parts {
		checkDir := filepath.Join(s.root, filepath.Join(parts[:i+1]...))
		if m, ok := s.ignores[filepath.Dir(checkDir)]; ok {
			name := parts[i]
			matched := false
			negate := false
			for _, p := range m.patterns {
				if p.match(name, isDir && i == len(parts)-1) {
					matched = true
					negate = p.negate
				}
			}
			if matched {
				return !negate
			}
		}
	}
	return false
}

func (s *scanner) scanDir(dir string, results *[]CommentItem) error {
	if s.fileCount >= s.maxFiles {
		return nil
	}
	s.loadGitignore(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		name := entry.Name()
		relPath, err := filepath.Rel(s.root, filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if relPath == "." {
			continue
		}
		if s.isIgnored(relPath, entry.IsDir()) {
			continue
		}
		fullPath := filepath.Join(dir, name)
		if entry.IsDir() {
			if name == ".git" || name == "node_modules" || name == "target" ||
				name == "vendor" || name == "dist" || name == "build" ||
				name == ".venv" || name == "__pycache__" || name == ".tox" ||
				name == ".next" || name == ".nuxt" || strings.HasPrefix(name, ".") {
				continue
			}
			if err := s.scanDir(fullPath, results); err != nil {
				return err
			}
		} else {
			if s.fileCount >= s.maxFiles {
				return nil
			}
			s.fileCount++
			ext := strings.ToLower(filepath.Ext(name))
			if !codeExts[ext] && name != "Dockerfile" && name != "Makefile" && name != "makefile" {
				continue
			}
			items, err := scanFile(fullPath)
			if err != nil {
				continue
			}
			rel, err := filepath.Rel(s.root, fullPath)
			if err != nil {
				rel = fullPath
			}
			for _, item := range items {
				item.File = rel
				*results = append(*results, item)
			}
		}
	}
	return nil
}

var lineCommentPatterns = []struct {
	prefix string
	suffix string
}{
	{"//", ""},
	{"#", ""},
	{"--", ""},
	{";", ""},
	{"%", ""},
	{"<!--", "-->"},
	{"/*", "*/"},
	{"{-", "-}"},
}

func commentPrefix(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".rs", ".java", ".c", ".h", ".cpp",
		".hpp", ".cs", ".swift", ".kt", ".scala", ".php", ".css", ".scss", ".less",
		".sql":
		return "//"
	case ".py", ".rb", ".sh", ".bash", ".zsh", ".pl", ".pm", ".lua", ".makefile",
		".yaml", ".yml", ".toml":
		return "#"
	case ".md", ".html", ".xml":
		return "<!--"
	case ".hs", ".lhs":
		return "--"
	case ".clj", ".cljs", ".edn":
		return ";"
	case ".tex":
		return "%"
	}
	if filename == "Dockerfile" || filename == "Makefile" || filename == "makefile" {
		return "#"
	}
	return ""
}

func scanFile(path string) ([]CommentItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var items []CommentItem
	scanner := bufio.NewScanner(f)
	lineNum := 0
	prefix := commentPrefix(filepath.Base(path))
	multiLine := false
	multiBuf := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.HasPrefix(line, "<!--") {
			if strings.Contains(line, "-->") {
			} else {
				multiBuf = line
				multiLine = true
			}
		}
		if multiLine {
			multiBuf += "\n" + line
			if strings.Contains(line, "-->") {
				multiLine = false
				line = multiBuf
			} else {
				continue
			}
		}

		commentText := ""
		if prefix == "<!--" {
			if m := regexp.MustCompile(`<!--\s*(.*?)-->`).FindStringSubmatch(line); m != nil {
				commentText = m[1]
			} else {
				continue
			}
		} else if prefix == "//" {
			idx := strings.Index(line, "//")
			if idx >= 0 {
				commentText = line[idx+2:]
			} else if m := regexp.MustCompile(`/\*\s*(.*?)\*/`).FindStringSubmatch(line); m != nil {
				commentText = m[1]
			} else {
				continue
			}
		} else if prefix != "" {
			idx := strings.Index(line, prefix)
			if idx < 0 {
				continue
			}
			commentText = line[idx+len(prefix):]
		} else {
			continue
		}

		matches := commentPattern.FindStringSubmatch(commentText)
		if matches == nil {
			continue
		}
		kind := strings.ToUpper(matches[1])
		text := strings.TrimSpace(matches[2])
		if text == "" {
			continue
		}
		items = append(items, CommentItem{
			File:     path,
			Line:     lineNum,
			Text:     text,
			Kind:     kind,
			Original: line,
		})
	}
	return items, scanner.Err()
}

func FindComments(rootDir string) ([]CommentItem, error) {
	fi, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", rootDir, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", rootDir)
	}
	s := newScanner(rootDir)
	var results []CommentItem
	if err := s.scanDir(rootDir, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// DONE: add support for multi-line block comments
// DONE: symlinks can cause infinite loops
// DONE: using strings.Contains as a quick check
