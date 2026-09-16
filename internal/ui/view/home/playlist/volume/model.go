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
	app   *core.App
}

func NewModel(label string, appRef *core.App, th theme.Theme) *Model {
	styles := newStyles(th)
	pr := progress.New(
		progress.WithColors(styles.volumeBarLow, styles.volumeBarHigh),
	)
	return &Model{
		bar:   pr,
		label: label,
		app:   appRef,
	}
}

func (m *Model) UpdateSize(width int) {
	m.width = width
	m.bar.SetWidth(m.width - lipgloss.Width(m.label))
}

func (m *Model) View() string {
	vol := m.app.State().Volume
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
