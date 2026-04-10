package playlist

import (
	"fmt"
	"path/filepath"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
)

type statusModel struct {
	label  string
	app    *core.App
	width  int
	styles styles
}

func newStatusModel(label string, appRef *core.App, styles styles) *statusModel {
	return &statusModel{
		label:  label,
		app:    appRef,
		styles: styles,
	}
}

func (m *statusModel) UpdateSize(width int) {
	m.width = width
}

func (m *statusModel) View() string {
	appState := m.app.State()
	playState := appState.PlayState
	truncateLeft := truncate.Left{}.MaxWidth(m.width)
	if playState == core.PlaybackStopped {
		text := m.label + m.styles.statusNone.Render("none")
		return truncateLeft.Render(text)
	}
	elapsed := appState.Elapsed()
	track := filepath.Base(appState.PlayTrack)
	if appState.Playing >= 0 && appState.Playing < len(appState.Playlist) {
		track = appState.Playlist[appState.Playing].DisplayName()
	}
	duration := fmt.Sprintf(" [%s]", formatDuration(elapsed))
	if appState.PlayDuration > 0 {
		duration = fmt.Sprintf(" [%s/%s]", formatDuration(elapsed), formatDuration(appState.PlayDuration))
	}
	stateStyle := playStateStyle(appState, m.styles)
	text := m.label + stateStyle.Render(track) + m.styles.statusTime.Render(duration)
	return truncateLeft.Render(text)
}

func playStateStyle(state core.State, styles styles) lipgloss.Style {
	switch state.PlayState {
	case core.PlaybackPaused:
		return styles.statusPause
	case core.PlaybackPlaying:
		return styles.statusPlay
	default:
		return styles.statusStop
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Seconds())
	minute := sec / 60
	sec = sec % 60
	return fmt.Sprintf("%02d:%02d", minute, sec)
}
