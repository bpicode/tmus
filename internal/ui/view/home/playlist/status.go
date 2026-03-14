package playlist

import (
	"fmt"
	"path/filepath"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/util"
)

type statusModel struct {
	label string
	app   *core.App
	width int
}

func newStatusModel(label string, appRef *core.App) *statusModel {
	return &statusModel{
		label: label,
		app:   appRef,
	}
}

func (m *statusModel) UpdateSize(width int) {
	m.width = width
}

func (m *statusModel) View() string {
	appState := m.app.State()
	playState := appState.PlayState
	if playState == core.PlaybackStopped {
		label := m.fmtLabel()
		stateText := "none"
		plain := label + stateText
		if m.width > 0 && len(plain) > m.width {
			return util.TruncateLeft(plain, m.width)
		}
		return label + styleStatusNone.Render(stateText)
	}
	elapsed := appState.Elapsed()
	track := filepath.Base(appState.PlayTrack)
	if appState.Playing >= 0 && appState.Playing < len(appState.Playlist) {
		track = appState.Playlist[appState.Playing].DisplayName()
	}
	label := m.fmtLabel()
	prefix := label
	suffix := fmt.Sprintf(" [%s]", formatDuration(elapsed))
	if appState.PlayDuration > 0 {
		suffix = fmt.Sprintf(" [%s/%s]", formatDuration(elapsed), formatDuration(appState.PlayDuration))
	}
	avail := m.width - len(prefix) - len(suffix)
	if avail < 1 {
		return util.TruncateLeft(prefix+track+suffix, m.width)
	}
	track = util.TruncateLeft(track, avail)
	stateStyle := playStateStyle(appState)
	return label + stateStyle.Render(track) + styleStatusTime.Render(suffix)
}

func playStateStyle(state core.State) lipgloss.Style {
	switch state.PlayState {
	case core.PlaybackPaused:
		return styleStatusPause
	case core.PlaybackPlaying:
		return styleStatusPlay
	default:
		return styleStatusStop
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

func (m *statusModel) fmtLabel() string {
	if m.width < 1 {
		return ""
	}
	if lipgloss.Width(m.label) >= m.width {
		return "" // give everything to the rest
	}
	return m.label
}
