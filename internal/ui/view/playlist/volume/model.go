package volume

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type Model struct {
	bar   progress.Model
	width int
	label string
}

func NewModel(label string, th theme.Theme) *Model {
	styles := newStyles(th)
	pr := progress.New(
		progress.WithColors(styles.volumeBarLow, styles.volumeBarHigh),
	)
	return &Model{
		bar:   pr,
		label: label,
	}
}

func (m *Model) UpdateSize(width int) {
	m.width = width
	m.bar.SetWidth(m.width - lipgloss.Width(m.label))
}

// Height returns the number of rows occupied by the volume display.
func (m *Model) Height() int {
	return 1
}

// View renders the supplied volume percentage.
func (m *Model) View(vol int) string {
	volPct := float64(vol) / float64(core.VolumeMax-core.VolumeMin)
	return m.fmtLabel() + m.bar.ViewAs(volPct)
}

func (m *Model) fmtLabel() string {
	if m.width < 1 {
		return ""
	}
	if lipgloss.Width(m.label) >= m.width {
		return "" // give everything to the bar
	}
	return m.label
}
