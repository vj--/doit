package tui

import (
	"hash/fnv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// tagPalette is the set of colors used to stably hash unknown tags into a
// consistent accent. Ordered so earlier entries are the most visually distinct.
var tagPalette = []lipgloss.AdaptiveColor{
	{Light: "#0891B2", Dark: "#5EEAD4"}, // cyan
	{Light: "#6D3FE0", Dark: "#A78BFA"}, // purple
	{Light: "#D97706", Dark: "#FBBF24"}, // amber
	{Light: "#059669", Dark: "#6EE7B7"}, // emerald
	{Light: "#DB2777", Dark: "#F9A8D4"}, // pink
	{Light: "#2563EB", Dark: "#93C5FD"}, // blue
	{Light: "#7C3AED", Dark: "#C4B5FD"}, // violet
	{Light: "#B91C1C", Dark: "#FCA5A5"}, // red
}

// knownTags maps conventional labels to fixed colors + a priority rank for
// the card stripe (lower rank = higher priority).
type tagMeta struct {
	color    lipgloss.AdaptiveColor
	priority int // 0 = highest; used to pick the card stripe color
}

var knownTags = map[string]tagMeta{
	"urgent":    {color: lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}, priority: 0},
	"blocked":   {color: lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}, priority: 1},
	"bug":       {color: lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}, priority: 2},
	"regression": {color: lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}, priority: 2},
	"feature":   {color: lipgloss.AdaptiveColor{Light: "#059669", Dark: "#6EE7B7"}, priority: 5},
	"docs":      {color: lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#93C5FD"}, priority: 6},
	"chore":     {color: lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}, priority: 9},
	"wip":       {color: lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#C4B5FD"}, priority: 4},
}

func tagColor(label string) lipgloss.AdaptiveColor {
	key := strings.ToLower(strings.TrimSpace(label))
	if m, ok := knownTags[key]; ok {
		return m.color
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return tagPalette[int(h.Sum32())%len(tagPalette)]
}

// stripeColorFor picks the accent for a focused card's left stripe: the
// highest-priority known tag wins, else the first tag's hashed color, else
// the default pink accent.
func stripeColorFor(labels []string) lipgloss.TerminalColor {
	best := -1
	var bestColor lipgloss.AdaptiveColor
	for _, l := range labels {
		key := strings.ToLower(strings.TrimSpace(l))
		if m, ok := knownTags[key]; ok {
			if best == -1 || m.priority < best {
				best = m.priority
				bestColor = m.color
			}
		}
	}
	if best != -1 {
		return bestColor
	}
	if len(labels) > 0 {
		return tagColor(labels[0])
	}
	return colPink
}

// labelPill renders a label with its tag-specific color.
func labelPill(label string) string {
	c := tagColor(label)
	return lipgloss.NewStyle().
		Foreground(c).
		Bold(true).
		Render("#" + label)
}
