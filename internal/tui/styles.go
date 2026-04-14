package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	colPink    = lipgloss.AdaptiveColor{Light: "#D4337F", Dark: "#FF5FD1"}
	colPurple  = lipgloss.AdaptiveColor{Light: "#6D3FE0", Dark: "#A78BFA"}
	colCyan    = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#5EEAD4"}
	colMuted   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"}
	colFaint   = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
	colText    = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#E5E7EB"}
	colError   = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	colBarBg   = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#1F2937"}
	colBarText = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#E5E7EB"}
)

var (
	styleBoardTitle = lipgloss.NewStyle().
			Foreground(colPink).
			Bold(true)

	styleFilename = lipgloss.NewStyle().
			Foreground(colMuted).
			Italic(true)

	styleAccentBar = lipgloss.NewStyle().
			Foreground(colPink)

	styleColumnTitle = lipgloss.NewStyle().
				Foreground(colText).
				Bold(true)

	styleColumnTitleFocused = lipgloss.NewStyle().
				Foreground(colPink).
				Bold(true).
				Underline(true)

	styleColumnCount = lipgloss.NewStyle().
				Foreground(colMuted)

	styleColumnRule = lipgloss.NewStyle().
			Foreground(colFaint)

	styleColumnRuleFocused = lipgloss.NewStyle().
				Foreground(colPink)

	styleCardTitle = lipgloss.NewStyle().
			Foreground(colText)

	styleCardTitleFocused = lipgloss.NewStyle().
				Foreground(colText).
				Bold(true)

	styleCardPreview = lipgloss.NewStyle().
				Foreground(colMuted)

	styleCardStripe = lipgloss.NewStyle().
			Foreground(colPink)

	styleCardStripeIdle = lipgloss.NewStyle().
				Foreground(colFaint)

	styleLabel = lipgloss.NewStyle().
			Foreground(colPurple).
			Background(lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2A1F4A"}).
			Padding(0, 1)

	styleHelp = lipgloss.NewStyle().Foreground(colMuted)

	styleError = lipgloss.NewStyle().Foreground(colError).Bold(true)

	styleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colPurple).
			Padding(1, 2)

	styleModalTitle = lipgloss.NewStyle().
			Foreground(colPink).
			Bold(true)

	styleStatusBarBase = lipgloss.NewStyle().
				Foreground(colBarText).
				Background(colBarBg)

	styleStatusMode = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colPink).
			Bold(true).
			Padding(0, 1)

	styleStatusSaving = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colPurple).
				Bold(true).
				Padding(0, 1)

	styleStatusError = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colError).
				Bold(true).
				Padding(0, 1)

	styleStatusSegment = lipgloss.NewStyle().
				Foreground(colBarText).
				Background(colBarBg).
				Padding(0, 1)

	styleStatusSegmentMuted = lipgloss.NewStyle().
				Foreground(colMuted).
				Background(colBarBg).
				Padding(0, 1)
)
