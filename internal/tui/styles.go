package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SetTheme overrides lipgloss background detection. Accepts "light", "dark",
// or "" (auto-detect). Background detection breaks under tmux, so callers
// should plumb a user-supplied override through here.
func SetTheme(theme string) {
	switch strings.ToLower(theme) {
	case "light":
		lipgloss.SetHasDarkBackground(false)
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	}
}

func init() {
	SetTheme(os.Getenv("DOIT_THEME"))
}

var (
	colBorderFocused = lipgloss.AdaptiveColor{Light: "#C71585", Dark: "#FF5FD1"}
	colBorder        = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
	colAccent        = lipgloss.AdaptiveColor{Light: "#0E7C66", Dark: "#5EEAD4"}
	colMuted         = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colError         = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	colText          = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
)

var (
	styleColumn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1).
			Margin(0, 1)

	styleColumnFocused = styleColumn.Copy().BorderForeground(colBorderFocused)

	styleColumnTitle = lipgloss.NewStyle().
				Foreground(colAccent).
				Bold(true).
				Padding(0, 0, 1, 0)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colBorder).
			Foreground(colText).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	styleCardFocused = styleCard.Copy().
				BorderForeground(colBorderFocused).
				Bold(true)

	styleHelp = lipgloss.NewStyle().Foreground(colMuted)

	styleStatus = lipgloss.NewStyle().Foreground(colMuted).Padding(0, 1)

	styleError = lipgloss.NewStyle().Foreground(colError).Bold(true)

	styleHeader = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true).
			Padding(0, 1)
)
