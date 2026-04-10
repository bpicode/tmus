package truncate

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type Right struct {
	lipgloss.Style
}

func (r Right) MaxWidth(w int) Left {
	style := r.Style.MaxWidth(w).Transform(func(s string) string { return truncateRightMulti(s, w) })
	return Left{Style: style}
}

func truncateRightMulti(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	return strings.Join(lines, "\n")
}
