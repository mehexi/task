package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommentPrefix(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"main.go", "//"},
		{"main.py", "#"},
		{"main.js", "//"},
		{"main.ts", "//"},
		{"main.tsx", "//"},
		{"main.jsx", "//"},
		{"main.rs", "//"},
		{"main.java", "//"},
		{"main.c", "//"},
		{"main.cpp", "//"},
		{"main.h", "//"},
		{"main.hpp", "//"},
		{"main.cs", "//"},
		{"main.swift", "//"},
		{"main.kt", "//"},
		{"main.scala", "//"},
		{"main.php", "//"},
		{"main.css", "//"},
		{"main.scss", "//"},
		{"main.less", "//"},
		{"main.sql", "//"},
		{"main.rb", "#"},
		{"main.sh", "#"},
		{"main.bash", "#"},
		{"main.zsh", "#"},
		{"main.pl", "#"},
		{"main.pm", "#"},
		{"main.lua", "#"},
		{"main.yaml", "#"},
		{"main.yml", "#"},
		{"main.toml", "#"},
		{"main.md", "<!--"},
		{"main.html", "<!--"},
		{"main.xml", "<!--"},
		{"main.hs", "--"},
		{"main.clj", ";"},
		{"main.tex", "%"},
		{"Dockerfile", "#"},
		{"Makefile", "#"},
		{"makefile", "#"},
		{"unknown.xyz", ""},
		{"noext", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := commentPrefix(tt.filename)
			if got != tt.expected {
				t.Errorf("commentPrefix(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		name     string
		expected bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.py", false},
		{"foo", "foo", true},
		{"foo", "foobar", false},
		{"foo/*", "foo/bar", true},
		{"foo/*", "bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.name, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.name)
			if got != tt.expected {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.expected)
			}
		})
	}
}

func TestParseGitignoreLine(t *testing.T) {
	tests := []struct {
		line    string
		wantOk  bool
		wantNeg bool
		wantDir bool
		wantAnc bool
		wantRaw string
	}{
		{"*.log", true, false, false, false, "*.log"},
		{"!important.go", true, true, false, false, "important.go"},
		{"build/", true, false, true, false, "build"},
		{"/secret", true, false, false, true, "secret"},
		{"", false, false, false, false, ""},
		{"# comment", false, false, false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			p, ok := parseGitignoreLine(tt.line)
			if ok != tt.wantOk {
				t.Errorf("parseGitignoreLine(%q) ok = %v, want %v", tt.line, ok, tt.wantOk)
			}
			if ok {
				if p.negate != tt.wantNeg {
					t.Errorf("parseGitignoreLine(%q) negate = %v, want %v", tt.line, p.negate, tt.wantNeg)
				}
				if p.dirOnly != tt.wantDir {
					t.Errorf("parseGitignoreLine(%q) dirOnly = %v, want %v", tt.line, p.dirOnly, tt.wantDir)
				}
				if p.anchored != tt.wantAnc {
					t.Errorf("parseGitignoreLine(%q) anchored = %v, want %v", tt.line, p.anchored, tt.wantAnc)
				}
				if p.raw != tt.wantRaw {
					t.Errorf("parseGitignoreLine(%q) raw = %q, want %q", tt.line, p.raw, tt.wantRaw)
				}
			}
		})
	}
}

func TestFindComments(t *testing.T) {
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "main.go")
	content := `package main

func main() {
	// TODO: implement error handling
	// FIXME: this is broken
	// NOTE: remember to refactor
	// HACK: temporary workaround
	// XXX: urgent
	// normal comment
	println("hello")
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"TODO": false,
		"FIXME": false,
		"NOTE": false,
		"HACK": false,
		"XXX":  false,
	}

	for _, item := range items {
		if _, ok := expected[item.Kind]; ok {
			expected[item.Kind] = true
		}
	}

	for kind, found := range expected {
		if !found {
			t.Errorf("comment kind %q not found", kind)
		}
	}
}

func TestFindComments_SkipsGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "vendor")
	os.MkdirAll(subDir, 0755)

	vendorFile := filepath.Join(subDir, "vendor.go")
	os.WriteFile(vendorFile, []byte("// TODO: vendor stuff\n"), 0644)

	srcFile := filepath.Join(tmpDir, "app.go")
	os.WriteFile(srcFile, []byte("// TODO: app stuff\n"), 0644)

	gitignore := filepath.Join(tmpDir, ".gitignore")
	os.WriteFile(gitignore, []byte("vendor/\n"), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if item.File == "vendor/vendor.go" {
			t.Errorf("found comment in ignored vendor directory")
		}
	}

	foundApp := false
	for _, item := range items {
		if item.File == "app.go" {
			foundApp = true
			break
		}
	}
	if !foundApp {
		t.Errorf("expected to find comment in app.go")
	}
}

func TestFindComments_SkipsNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	nodeDir := filepath.Join(tmpDir, "node_modules")
	os.MkdirAll(nodeDir, 0755)
	os.WriteFile(filepath.Join(nodeDir, "lib.js"), []byte("// TODO: something\n"), 0644)

	srcFile := filepath.Join(tmpDir, "index.js")
	os.WriteFile(srcFile, []byte("// TODO: main thing\n"), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if item.File == "node_modules/lib.js" {
			t.Errorf("found comment in node_modules")
		}
	}

	found := false
	for _, item := range items {
		if item.File == "index.js" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find comment in index.js")
	}
}

func TestScanFile_NoComments(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clean.go")
	os.WriteFile(tmpFile, []byte("package main\nfunc main() {}\n"), 0644)

	items, err := scanFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestScanFile_NonExistent(t *testing.T) {
	_, err := scanFile("/nonexistent/file.go")
	if err == nil {
		t.Errorf("expected error for non-existent file")
	}
}

func TestFindComments_NotADirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.go")
	os.WriteFile(tmpFile, []byte("package main\n"), 0644)

	_, err := FindComments(tmpFile)
	if err == nil {
		t.Errorf("expected error when path is not a directory")
	}
}

func TestFindComments_NonExistentDir(t *testing.T) {
	_, err := FindComments("/nonexistent/dir")
	if err == nil {
		t.Errorf("expected error for non-existent directory")
	}
}

func TestFindComments_PythonAndRuby(t *testing.T) {
	tmpDir := t.TempDir()

	pyFile := filepath.Join(tmpDir, "app.py")
	os.WriteFile(pyFile, []byte("# TODO: add validation\n# FIXME: slow query\ndef foo():\n    pass\n"), 0644)

	rbFile := filepath.Join(tmpDir, "app.rb")
	os.WriteFile(rbFile, []byte("# TODO: refactor this\nclass Foo\nend\n"), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 comments, got %d", len(items))
	}
}

func TestFindComments_HTMLBlockComments(t *testing.T) {
	tmpDir := t.TempDir()

	htmlFile := filepath.Join(tmpDir, "index.html")
	os.WriteFile(htmlFile, []byte("<!-- TODO: update title -->\n<html><body></body></html>\n"), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(items))
	}
	if items[0].Kind != "TODO" {
		t.Errorf("expected TODO kind, got %s", items[0].Kind)
	}
}

func TestFindComments_MultiLineHTML(t *testing.T) {
	tmpDir := t.TempDir()

	htmlFile := filepath.Join(tmpDir, "page.html")
	content := "<!--\nNOT allowed (no prefix on first line)\n-->\n"
	os.WriteFile(htmlFile, []byte(content), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 comments for multi-line without prefix, got %d", len(items))
	}
}

func TestFindComments_CStyleBlock(t *testing.T) {
	tmpDir := t.TempDir()

	cssFile := filepath.Join(tmpDir, "style.css")
	os.WriteFile(cssFile, []byte("/* TODO: refactor layout */\nbody { margin: 0; }\n"), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(items))
	}
	if items[0].Kind != "TODO" {
		t.Errorf("expected TODO, got %s", items[0].Kind)
	}
}

func TestFindComments_LanguageCommentPrefixes(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"script.py": "# TODO: python comment\n",
		"styles.css": "/* TODO: css comment */\n",
		"Makefile":  "# TODO: make target\n",
	}

	for name, content := range files {
		os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
	}

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != len(files) {
		t.Errorf("expected %d comments, got %d", len(files), len(items))
	}
}

func TestGitignoreMatcher(t *testing.T) {
	tmpDir := t.TempDir()
	gitignore := filepath.Join(tmpDir, ".gitignore")
	os.WriteFile(gitignore, []byte("*.go\n!important.go\nbuild/\n"), 0644)

	os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte("// TODO: fix\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "important.go"), []byte("// TODO: important\n"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "build"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "build", "output.go"), []byte("// TODO: build output\n"), 0644)

	items, err := FindComments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if item.File == "app.go" {
			t.Errorf("app.go should be ignored by *.go pattern")
		}
		if item.File == "build/output.go" {
			t.Errorf("build/output.go should be ignored by build/ pattern")
		}
	}

	foundImportant := false
	for _, item := range items {
		if item.File == "important.go" {
			foundImportant = true
			break
		}
	}
	if !foundImportant {
		t.Errorf("important.go should NOT be ignored (negated pattern)")
	}
}

func TestFindComments_MaxFilesLimit(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 100; i++ {
		f, err := os.Create(filepath.Join(tmpDir, "file_"+string(rune('a'+i%26))+string(rune('0'+i/10))+".go"))
		if err == nil {
			f.Write([]byte("// TODO: file\n"))
			f.Close()
		}
	}

	s := newScanner(tmpDir)
	s.maxFiles = 10
	var results []CommentItem
	err := s.scanDir(tmpDir, &results)
	if err != nil {
		t.Fatal(err)
	}
	if s.fileCount > 10 {
		t.Errorf("fileCount %d exceeded maxFiles %d", s.fileCount, s.maxFiles)
	}
}
