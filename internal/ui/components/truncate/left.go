package truncate

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type Left struct {
	lipgloss.Style
}

func (l Left) MaxWidth(w int) Left {
	style := l.Style.MaxWidth(w).Transform(func(s string) string { return truncateLeftMulti(s, w) })
	return Left{Style: style}
}

func truncateLeftMulti(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = truncateLeftSingle(lines[i], w)
	}
	return strings.Join(lines, "\n")
}

func truncateLeftSingle(s string, w int) string {
	if w <= 0 {
		return ""
	}
	width := lipgloss.Width(s)
	if width <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	toRemove := width - (w - 1)
	return ansi.TruncateLeft(s, toRemove, "…")
}
