package lyrics

import (
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/lyrics"
	"github.com/bpicode/tmus/internal/ui/components/error_view"
	"github.com/charmbracelet/x/ansi"
)

type Model struct {
	show           bool
	width          int
	height         int
	lyricsViewport viewport.Model
	trackID        uint64
	trackPath      string
	followPlay     bool
	followLine     bool
	loading        bool
	data           lyrics.Lyrics
	app            *core.App
	errorView      *error_view.Model
}

func NewModel(app *core.App) *Model {
	vp := viewport.New()
	vp.LeftGutterFunc = viewport.NoGutter
	return &Model{
		lyricsViewport: vp,
		app:            app,
		followLine:     true,
		errorView: error_view.New(error_view.Styles{
			ErrorStyle:          styleError,
			UnwrappedErrorStyle: styleError,
		}),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) UpdateSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
}

func (m *Model) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.show {
		return nil, false
	}
	switch msg.String() {
	case "q", "esc", "L":
		m.Show(false)
		return nil, true
	case "f":
		m.followLine = !m.followLine
		return nil, true
	case "up", "k":
		m.lyricsViewport.ScrollUp(1)
		return nil, true
	case "down", "j":
		m.lyricsViewport.ScrollDown(1)
		return nil, true
	case "pgup", "pageup":
		m.lyricsViewport.PageUp()
		return nil, true
	case "pgdown", "pagedown":
		m.lyricsViewport.PageDown()
		return nil, true
	case "home", "pos1":
		m.lyricsViewport.GotoTop()
		return nil, true
	case "end":
		m.lyricsViewport.GotoBottom()
		return nil, true
	default:
		return nil, false
	}
}

func (m *Model) HandleEvent(event core.LyricsEvent) bool {
	if !m.show {
		return false
	}
	if event.TrackID != m.trackID {
		return false
	}
	if event.Path != m.trackPath {
		return false
	}
	m.loading = false
	if event.Err != nil {
		m.errorView.SetErr(event.Err)
		return true
	}
	m.errorView.SetErr(nil)
	m.data = event.Lyrics
	return true
}

func (m *Model) SyncState() {
	if !m.show || !m.followPlay {
		return
	}
	state := m.app.State()
	track, ok := lyricsPlayingTrack(state)
	if !ok || track.ID == 0 || track.Path == "" {
		return
	}
	if track.ID == m.trackID && track.Path == m.trackPath {
		return
	}
	m.trackID = track.ID
	m.trackPath = track.Path
	m.loading = true
	m.errorView.SetErr(nil)
	m.lyricsViewport.GotoTop()
	m.data = lyrics.Lyrics{}
	_ = m.app.Dispatch(core.Command{
		Type:    core.CmdRequestLyrics,
		TrackID: track.ID,
		Path:    track.Path,
	})
}

func (m *Model) Show(show bool) {
	if show {
		state := m.app.State()
		track, index, ok := lyricsTrackForOpen(state)
		if !ok {
			return
		}
		if track.ID == 0 || track.Path == "" {
			return
		}
		m.show = true
		m.trackID = track.ID
		m.trackPath = track.Path
		m.followPlay = index == state.Playing && index >= 0
		m.loading = true
		m.errorView.SetErr(nil)
		m.lyricsViewport.GotoTop()
		m.data = lyrics.Lyrics{}
		cmd := core.Command{Type: core.CmdRequestLyrics, TrackID: track.ID, Path: track.Path}
		_ = m.app.Dispatch(cmd)
	} else {
		m.show = false
		m.trackID = 0
		m.trackPath = ""
		m.followPlay = false
		m.loading = false
		m.errorView.SetErr(nil)
		m.lyricsViewport.GotoTop()
		m.data = lyrics.Lyrics{}
	}
}

func (m *Model) Visible() bool {
	return m.show
}

func (m *Model) View() string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	state := m.app.State()
	innerWidth, innerHeight := m.innerSize()

	availableWidth := innerWidth
	viewportHeight := max(innerHeight-3, 0) // 3 -> 1 for title, 1 for track, 1 for empty line after track
	m.lyricsViewport.SetWidth(availableWidth)
	m.lyricsViewport.SetHeight(viewportHeight)

	title := styleTitle.Render(ansi.Truncate("📜 Lyrics", availableWidth, "…"))
	trackName := sanitizeTerminalText(displayNameForTrack(state, m.trackID, m.trackPath))
	track := styleTrack.Render(ansi.Truncate(trackName, availableWidth, "…"))
	pad := ""
	headers := strings.Join([]string{title, track, pad}, "\n")

	lines, hightlightIndex := m.bodyLines(availableWidth, state)
	m.lyricsViewport.SetContentLines(lines)
	if hightlightIndex >= 0 && hightlightIndex < len(lines) && m.followLine {
		highlightLine := lines[hightlightIndex]
		m.lyricsViewport.EnsureVisible(hightlightIndex, 0, lipgloss.Width(highlightLine))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, headers, m.lyricsViewport.View())
	inner := lipgloss.NewStyle().MaxWidth(availableWidth).MaxHeight(innerHeight).Render(content)
	styled := styleOverlay.Render(inner)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, styled)
}

func (m *Model) innerSize() (int, int) {
	contentWidth := max(m.width-styleOverlay.GetHorizontalFrameSize(), 0)
	contentHeight := max(m.height-styleOverlay.GetVerticalFrameSize(), 0)
	return contentWidth, contentHeight
}

func displayNameForTrack(state core.State, trackID uint64, path string) string {
	for _, track := range state.Playlist {
		if track.ID != trackID {
			continue
		}
		if display := track.DisplayName(); display != "" {
			return display
		}
		if track.Path != "" {
			return filepath.Base(track.Path)
		}
		break
	}
	if path != "" {
		return filepath.Base(path)
	}
	return ""
}

func (m *Model) bodyLines(maxWidth int, state core.State) ([]string, int) {
	if maxWidth < 1 {
		return nil, -1
	}

	truncate := func(s string) string {
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, maxWidth, "…")
		}
		return strings.Join(lines, "\n")
	}

	switch {
	case m.loading:
		return []string{truncate(styleTrack.Render("Loading..."))}, -1
	case m.errorView.HasErr():
		lines := []string{
			truncate(m.errorView.View()),
		}
		return lines, -1
	case len(m.data.Lines) == 0:
		return []string{truncate(styleEmpty.Render("No lyrics available"))}, -1
	default:
		active := -1
		if m.data.Timed && m.followPlay && m.matchesPlaying(state) {
			active = activeLyricIndex(m.data.Lines, state.Elapsed())
		}
		return lyricsLinesForWidth(m.data.Lines, maxWidth, active), active
	}
}

func (m *Model) matchesPlaying(state core.State) bool {
	track, ok := lyricsPlayingTrack(state)
	if !ok {
		return false
	}
	if track.ID != 0 && m.trackID != 0 && track.ID != m.trackID {
		return false
	}
	if track.Path != "" && m.trackPath != "" && track.Path != m.trackPath {
		return false
	}
	return true
}

func activeLyricIndex(lines []lyrics.Line, elapsed time.Duration) int {
	active := -1
	for i, line := range lines {
		if !line.HasTime {
			continue
		}
		if elapsed >= line.Time {
			active = i
		}
	}
	return active
}

func lyricsLinesForWidth(lines []lyrics.Line, width int, active int) []string {
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		trimmed := ansi.Truncate(sanitizeTerminalText(line.Text), width, "…")
		if i == active {
			out = append(out, styleActiveLine.Render(trimmed))
		} else {
			out = append(out, trimmed)
		}
	}
	return out
}

func sanitizeTerminalText(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\t", "    ")
	value = ansi.Strip(value)
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

func lyricsTrackForOpen(state core.State) (core.Track, int, bool) {
	if len(state.Playlist) == 0 {
		return core.Track{}, -1, false
	}
	if state.Cursor >= 0 && state.Cursor < len(state.Playlist) {
		return state.Playlist[state.Cursor], state.Cursor, true
	}
	if state.Playing >= 0 && state.Playing < len(state.Playlist) {
		return state.Playlist[state.Playing], state.Playing, true
	}
	return core.Track{}, -1, false
}

func lyricsPlayingTrack(state core.State) (core.Track, bool) {
	if state.Playing < 0 || state.Playing >= len(state.Playlist) {
		return core.Track{}, false
	}
	return state.Playlist[state.Playing], true
}
