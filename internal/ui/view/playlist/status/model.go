package status

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type Model struct {
	label  string
	app    *core.App
	width  int
	styles styles
}

func NewModel(label string, appRef *core.App, th theme.Theme) *Model {
	return &Model{
		label:  label,
		app:    appRef,
		styles: newStyles(th),
	}
}

func (m *Model) UpdateSize(width int) {
	m.width = width
}

// Height returns the number of rows occupied by the status.
func (m *Model) Height() int {
	if m.width < 1 {
		return 0
	}
	return 1
}

func (m *Model) View() string {
	appState := m.app.State()
	playState := appState.Playback.State
	truncateLeft := truncate.Left{}.MaxWidth(m.width)
	if playState == core.PlaybackStopped {
		text := m.label + m.styles.statusNone.Render("none")
		return truncateLeft.Render(text)
	}
	elapsed := appState.Elapsed()
	track := sanitize.TerminalText(filepath.Base(appState.Playback.Source))
	if appState.Playing >= 0 && appState.Playing < len(appState.Playlist) {
		track = sanitize.TerminalText(appState.Playlist[appState.Playing].DisplayName())
	}
	if nowPlaying := appState.Playback.NowPlaying.DisplayName(); nowPlaying != "" {
		track = sanitize.TerminalText(nowPlaying)
	}
	duration := fmt.Sprintf(" [%s]", formatDuration(elapsed))
	if appState.Playback.Duration > 0 {
		duration = fmt.Sprintf(" [%s/%s]", formatDuration(elapsed), formatDuration(appState.Playback.Duration))
	}
	stateStyle := m.styles.playStateStyle(appState)
	text := m.label + stateStyle.Render(track) + m.styles.statusTime.Render(duration)
	return truncateLeft.Render(text)
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
