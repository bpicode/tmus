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
	"github.com/bpicode/tmus/internal/ui/components/errorview"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
	"github.com/bpicode/tmus/internal/ui/theme"
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
	errorView      *errorview.Model
	styles         styles
}

type Config struct {
	Theme      theme.Theme
	FollowLine bool
	App        *core.App
}

func NewModel(cfg Config) *Model {
	vp := viewport.New()
	vp.LeftGutterFunc = viewport.NoGutter
	styles := newStyles(cfg.Theme)
	return &Model{
		lyricsViewport: vp,
		app:            cfg.App,
		followLine:     cfg.FollowLine,
		styles:         styles,
		errorView:      errorview.New(errorview.Styles{Error: styles.err}),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
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

	title := truncate.Right{Style: m.styles.title}.MaxWidth(availableWidth).Render("📜 Lyrics")
	trackName := sanitizeTerminalText(displayNameForTrack(state, m.trackID, m.trackPath))
	track := truncate.Right{Style: m.styles.track}.MaxWidth(availableWidth).Render(trackName)
	pad := ""
	headers := strings.Join([]string{title, track, pad}, "\n")

	lines, highlightIndex := m.bodyLines(availableWidth, state)
	m.lyricsViewport.SetContentLines(lines)
	if highlightIndex >= 0 && highlightIndex < len(lines) && m.followLine {
		highlightLine := lines[highlightIndex]
		m.lyricsViewport.EnsureVisible(highlightIndex, 0, lipgloss.Width(highlightLine))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, headers, m.lyricsViewport.View())
	inner := lipgloss.NewStyle().MaxWidth(availableWidth).MaxHeight(innerHeight).Render(content)
	styled := m.styles.overlay.Render(inner)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, styled)
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleSizeMsg(msg)
	case tea.KeyPressMsg:
		return m.handleKeyPressMsg(msg)
	case core.LyricsEvent:
		return m.handleLyricsEvent(msg)
	case core.StateEvent:
		return m.handleStateEvent()
	default:
		return m, nil, false
	}
}

func (m *Model) handleSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd, bool) {
	m.width = msg.Width
	m.height = msg.Height
	return m, nil, false
}

func (m *Model) handleKeyPressMsg(msg tea.KeyPressMsg) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}
	switch msg.String() {
	case "q", "esc", "L":
		m.Show(false)
		return m, nil, true
	case "f":
		m.followLine = !m.followLine
		return m, nil, true
	case "up", "k":
		m.lyricsViewport.ScrollUp(1)
		return m, nil, true
	case "down", "j":
		m.lyricsViewport.ScrollDown(1)
		return m, nil, true
	case "pgup", "pageup":
		m.lyricsViewport.PageUp()
		return m, nil, true
	case "pgdown", "pagedown":
		m.lyricsViewport.PageDown()
		return m, nil, true
	case "home", "pos1":
		m.lyricsViewport.GotoTop()
		return m, nil, true
	case "end":
		m.lyricsViewport.GotoBottom()
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m *Model) handleLyricsEvent(event core.LyricsEvent) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}
	if event.TrackID != m.trackID {
		return m, nil, false
	}
	if event.Path != m.trackPath {
		return m, nil, false
	}
	m.loading = false
	if event.Err != nil {
		m.errorView.SetErr(event.Err)
		return m, nil, false
	}
	m.errorView.SetErr(nil)
	m.data = event.Lyrics
	return m, nil, false
}

func (m *Model) handleStateEvent() (*Model, tea.Cmd, bool) {
	if !m.show || !m.followPlay {
		return m, nil, false
	}
	state := m.app.State()
	track, ok := lyricsPlayingTrack(state)
	if !ok || track.ID == 0 || track.Path == "" {
		return m, nil, false
	}
	if track.ID == m.trackID && track.Path == m.trackPath {
		return m, nil, false
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
	return m, nil, false
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

func (m *Model) FollowLine() bool {
	return m.followLine
}

func (m *Model) innerSize() (int, int) {
	contentWidth := max(m.width-m.styles.overlay.GetHorizontalFrameSize(), 0)
	contentHeight := max(m.height-m.styles.overlay.GetVerticalFrameSize(), 0)
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

	switch {
	case m.loading:
		style := truncate.Right{Style: m.styles.track}.MaxWidth(maxWidth)
		return []string{style.Render("Loading...")}, -1
	case m.errorView.HasErr():
		style := truncate.Right{}.MaxWidth(maxWidth)
		return []string{style.Render(m.errorView.View())}, -1
	case len(m.data.Lines) == 0:
		style := truncate.Right{Style: m.styles.empty}.MaxWidth(maxWidth)
		return []string{style.Render("No lyrics available")}, -1
	default:
		active := -1
		if m.data.Timed && m.followPlay && m.matchesPlaying(state) {
			active = activeLyricIndex(m.data.Lines, state.Elapsed())
		}
		return lyricsLinesForWidth(m.data.Lines, maxWidth, active, m.styles), active
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

func lyricsLinesForWidth(lines []lyrics.Line, width int, active int, styles styles) []string {
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		style := lipgloss.NewStyle()
		if i == active {
			style = styles.activeLine
		}
		truncateRight := truncate.Right{Style: style}.MaxWidth(width)
		out = append(out, truncateRight.Render(sanitizeTerminalText(line.Text)))
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
