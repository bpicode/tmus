package playlist

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
)

type volumeModel struct {
	bar    progress.Model
	width  int
	label  string
	app    *core.App
	styles styles
}

func newVolumeModel(label string, appRef *core.App, styles styles) *volumeModel {
	pr := progress.New(
		progress.WithColors(styles.volumeBarLow, styles.volumeBarHigh),
	)
	return &volumeModel{
		bar:    pr,
		label:  label,
		app:    appRef,
		styles: styles,
	}
}

func (m *volumeModel) UpdateSize(width int) {
	m.width = width
	m.bar.SetWidth(m.width - lipgloss.Width(m.label))
}

func (m *volumeModel) View() string {
	vol := m.app.State().Volume
	volPct := float64(vol) / float64(core.VolumeMax-core.VolumeMin)
	return m.fmtLabel() + m.bar.ViewAs(volPct)
}

func (m *volumeModel) fmtLabel() string {
	if m.width < 1 {
		return ""
	}
	if lipgloss.Width(m.label) >= m.width {
		return "" // give everything to the bar
	}
	return m.label
}
