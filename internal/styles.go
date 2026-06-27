package internal

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	bg   lipgloss.Color
	surf lipgloss.Color
	bord lipgloss.Color
	ac   lipgloss.Color
	grn  lipgloss.Color
	red  lipgloss.Color
	yel  lipgloss.Color
	mut  lipgloss.Color
	txt  lipgloss.Color
)

var (
	HeaderS  lipgloss.Style
	StatusS  lipgloss.Style

	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style

	TitleS lipgloss.Style

	ProjActiveS   lipgloss.Style
	ProjInactiveS lipgloss.Style

	DoneS lipgloss.Style
	TodoS lipgloss.Style

	PHighS lipgloss.Style
	PMedS  lipgloss.Style
	PLowS  lipgloss.Style

	LabelS lipgloss.Style
	ValueS lipgloss.Style

	ActiveProjBg lipgloss.Style

	DimS    lipgloss.Style
	AccentS lipgloss.Style
)

func setDefaultColors() {
	bg = "#1a1b26"
	surf = "#24283b"
	bord = "#414868"
	ac = "#7aa2f7"
	grn = "#9ece6a"
	red = "#f7768e"
	yel = "#e0af68"
	mut = "#565f89"
	txt = "#c0caf5"
}

func buildStyles() {
	HeaderS = lipgloss.NewStyle().
		Background(surf).
		Foreground(txt).
		Padding(0, 1)

	StatusS = lipgloss.NewStyle().
		Background(surf).
		Foreground(mut).
		Padding(0, 1)

	ActiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ac)

	InactiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bord)

	TitleS = lipgloss.NewStyle().
		Foreground(ac).
		Bold(true)

	ProjActiveS = lipgloss.NewStyle().
		Foreground(ac).
		Bold(true)

	ProjInactiveS = lipgloss.NewStyle().
		Foreground(txt)

	DoneS = lipgloss.NewStyle().
		Foreground(grn)

	TodoS = lipgloss.NewStyle().
		Foreground(mut)

	PHighS = lipgloss.NewStyle().
		Background(lipgloss.Color("#ff4444")).
		Foreground(bg).
		Bold(true).
		Padding(0, 1)

	PMedS = lipgloss.NewStyle().
		Background(lipgloss.Color("#ffaa00")).
		Foreground(bg).
		Bold(true).
		Padding(0, 1)

	PLowS = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1)

	LabelS = lipgloss.NewStyle().
		Foreground(mut)

	ValueS = lipgloss.NewStyle().
		Foreground(txt)

	ActiveProjBg = lipgloss.NewStyle().
		Background(surf)

	DimS = lipgloss.NewStyle().
		Foreground(mut)

	AccentS = lipgloss.NewStyle().
		Foreground(ac)
}

func omarchyColorsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
}

func parseColorsTOML(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	colors := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"")
		if strings.HasPrefix(val, "#") {
			colors[key] = val
		}
	}
	return colors, scanner.Err()
}

func applyThemeColors(colors map[string]string) bool {
	if colors == nil {
		return false
	}
	get := func(key, fallback string) lipgloss.Color {
		if v, ok := colors[key]; ok && v != "" {
			return lipgloss.Color(v)
		}
		return lipgloss.Color(fallback)
	}
	bg = get("background", "#1a1b26")
	surf = get("color0", "#24283b")
	bord = get("color8", "#414868")
	ac = get("accent", "#7aa2f7")
	grn = get("color2", "#9ece6a")
	red = get("color1", "#f7768e")
	yel = get("color3", "#e0af68")
	mut = get("color7", "#565f89")
	txt = get("foreground", "#c0caf5")
	return true
}

func LoadThemeColors() {
	path := omarchyColorsPath()
	if path != "" {
		colors, err := parseColorsTOML(path)
		if err == nil && applyThemeColors(colors) {
			buildStyles()
			return
		}
	}
	setDefaultColors()
	buildStyles()
}

func init() {
	LoadThemeColors()
}
