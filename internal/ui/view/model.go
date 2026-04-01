package view

import (
	"os"
	"path"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/archive"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view/help"
	"github.com/bpicode/tmus/internal/ui/view/home"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist"
	"github.com/bpicode/tmus/internal/ui/view/lyrics"
	"github.com/bpicode/tmus/internal/ui/view/track_info"
)

type Model struct {
	app       *core.App
	home      *home.Model
	help      *help.Model
	trackInfo *track_info.Model
	lyrics    *lyrics.Model
	events    eventChannels
	width     int
	height    int
}

type eventChannels struct {
	state         <-chan core.StateEvent
	unsubState    func()
	metadata      <-chan core.MetadataEvent
	unsubMetadata func()
	lyrics        <-chan core.LyricsEvent
	unsubLyrics   func()
}

func NewModel(appRef *core.App, startDir string, openFiles []string, cfg config.TUIConfig) *Model {
	st, err := loadState()
	if err != nil {
		st = State{}
	}

	cwd := startDir
	if cwd == "" && st.BrowserDir != "" {
		cwd = st.BrowserDir
	}
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	if !archive.IsArchivePath(cwd) {
		if abs, err := filepath.Abs(cwd); err == nil {
			cwd = abs
		}
		if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
			cwd = resolved
		}
	}

	th := theme.New(cfg.Theme)
	m := &Model{
		app:       appRef,
		home:      home.NewModel(home.Config{Cwd: cwd, HomeDir: cfg.BrowserHome, Theme: th, App: appRef}),
		help:      help.NewModel(th),
		trackInfo: track_info.NewModel(track_info.Config{Theme: th, App: appRef}),
		lyrics:    lyrics.NewModel(lyrics.Config{Theme: th, App: appRef}),
	}
	m.restore(st)
	m.openFiles(openFiles)

	return m
}

func (m *Model) Init() tea.Cmd {
	m.events.state, m.events.unsubState = m.app.SubscribeStateEvents()
	m.events.metadata, m.events.unsubMetadata = m.app.SubscribeMetadataEvents()
	m.events.lyrics, m.events.unsubLyrics = m.app.SubscribeLyricsEvents()
	m.home.Show(true)
	return tea.Batch(
		m.help.Init(),
		m.trackInfo.Init(),
		m.lyrics.Init(),
		m.home.Init(),
		m.listenForStateEvent(),
		m.listenForMetadataEvent(),
		m.listenForLyricsEvent(),
		tickCmd(), // UI updates every second to keep things like elapsed time or lyrics updated.
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmdSub tea.Cmd
	var stop bool

	m.help, cmdSub, stop = m.help.Update(msg)
	if stop {
		return m, cmdSub
	}
	cmds = append(cmds, cmdSub)

	m.lyrics, cmdSub, stop = m.lyrics.Update(msg)
	if stop {
		return m, cmdSub
	}
	cmds = append(cmds, cmdSub)

	m.trackInfo, cmdSub, stop = m.trackInfo.Update(msg)
	if stop {
		return m, cmdSub
	}
	cmds = append(cmds, cmdSub)

	m.home, cmdSub, stop = m.home.Update(msg)
	if stop {
		return m, cmdSub
	}
	cmds = append(cmds, cmdSub)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Shutdown()
			cmds = append(cmds, tea.Quit)
		}
		switch msg.Key().Text {
		case "?":
			if !m.trackInfo.Visible() && !m.lyrics.Visible() {
				m.help.Show(true)
			}
		}
	case core.StateEvent:
		cmds = append(cmds, m.listenForStateEvent())
	case core.MetadataEvent:
		cmds = append(cmds, m.listenForMetadataEvent())
	case core.LyricsEvent:
		cmds = append(cmds, m.listenForLyricsEvent())
	case playlist.ToggleLyricsMsg:
		if !m.trackInfo.Visible() && !m.help.Visible() {
			m.lyrics.Show(true)
		}
	case playlist.ToggleTrackInfoMsg:
		if !m.lyrics.Visible() && !m.help.Visible() {
			m.trackInfo.Show(true)
		}
	case tickMsg:
		cmds = append(cmds, tickCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() tea.View {
	content := ""
	if m.width == 0 {
		content = "loading..."
	} else if m.help.Visible() {
		content = m.help.View()
	} else if m.trackInfo.Visible() {
		content = m.trackInfo.View()
	} else if m.lyrics.Visible() {
		content = m.lyrics.View()
	} else {
		content = m.home.View()
	}

	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "tmus"
	return view
}

func (m *Model) restore(s State) {
	tracks := make([]core.Track, 0, len(s.Playlist))
	for _, entry := range s.Playlist {
		if entry.Path == "" {
			continue
		}
		name := entry.Name
		if name == "" {
			name = trackName(entry.Path)
		}
		tracks = append(tracks, core.Track{
			Name:     name,
			Path:     entry.Path,
			Artist:   entry.Artist,
			Title:    entry.Title,
			Album:    entry.Album,
			Duration: entry.Duration,
		})
	}
	cursor := s.Cursor
	if cursor < 0 || cursor >= len(tracks) {
		cursor = s.Playing
	}
	m.app.Restore(tracks, cursor, ParseQueueMode(s.QueueMode))
	volume := core.DefaultVolume
	if s.Volume != nil {
		volume = *s.Volume
	}
	m.app.SetVolume(volume)
	m.home.ShowBrowser(!s.BrowserHidden)
	switch s.Focus {
	case "playlist":
		m.home.FocusPlaylist()
	default:
		m.home.FocusBrowser()
	}
	if s.BrowserHidden {
		m.home.FocusPlaylist()
	}
}

func trackName(value string) string {
	if archive.IsArchivePath(value) {
		if _, archivePath, inner, err := archive.SplitPath(value); err == nil {
			if inner != "" {
				return path.Base(inner)
			}
			return filepath.Base(archivePath)
		}
	}
	return filepath.Base(value)
}

func (m *Model) openFiles(openFiles []string) {
	if len(openFiles) == 0 {
		return
	}
	startIndex := len(m.app.State().Playlist)
	tracks := make([]core.Track, 0, len(openFiles))
	for _, file := range openFiles {
		if file == "" {
			continue
		}
		cleanPath := filepath.Clean(file)
		if abs, err := filepath.Abs(cleanPath); err == nil {
			cleanPath = abs
		}
		if !library.IsAudio(cleanPath) {
			continue
		}
		tracks = append(tracks, core.Track{
			Name: trackName(cleanPath),
			Path: cleanPath,
		})
	}
	if len(tracks) == 0 {
		return
	}
	_ = m.app.Dispatch(core.Command{Type: core.CmdAddAll, Tracks: tracks})
	_ = m.app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: startIndex})
	_ = m.app.Dispatch(core.Command{Type: core.CmdPlayFromCursor})
}

func (m *Model) SaveState() error {
	statePath, err := DefaultPath()
	if err != nil {
		return err
	}
	appState := m.app.State()
	tracks := make([]Track, 0, len(appState.Playlist))
	for _, track := range appState.Playlist {
		tracks = append(tracks, Track{
			Path:     track.Path,
			Name:     track.Name,
			Artist:   track.Artist,
			Title:    track.Title,
			Album:    track.Album,
			Duration: track.Duration,
		})
	}
	focus := "browser"
	if m.home.PlaylistFocused() {
		focus = "playlist"
	}
	return Save(statePath, State{
		BrowserDir:    m.home.BrowserCwd(),
		BrowserHidden: m.home.BrowserHidden(),
		Focus:         focus,
		Volume:        new(appState.Volume),
		QueueMode:     QueueModeString(appState.QueueMode),
		Playlist:      tracks,
		Playing:       appState.Playing,
		Cursor:        appState.Cursor,
	})
}

func (m *Model) Shutdown() {
	if m.events.unsubState != nil {
		m.events.unsubState()
		m.events.unsubState = nil
	}
	if m.events.unsubMetadata != nil {
		m.events.unsubMetadata()
		m.events.unsubMetadata = nil
	}
	if m.events.unsubLyrics != nil {
		m.events.unsubLyrics()
		m.events.unsubLyrics = nil
	}
	m.app.Shutdown()
}

// stateClosedMsg is returned when the state event channel is closed.
// This prevents infinite loops in Bubble Tea's background goroutines
// where it would otherwise continuously read zero-values from a closed channel.
// It does not need to be handled explicitly.
type stateClosedMsg struct{}

// listenForStateEvent returns a tea.Cmd that listens for state events
// without blocking the main Update loop.
func (m *Model) listenForStateEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.state
		if !ok {
			return stateClosedMsg{}
		}
		return event
	}
}

// metadataClosedMsg is returned when the metadata event channel is closed.
// This prevents infinite loops in Bubble Tea's background goroutines
// where it would otherwise continuously read zero-values from a closed channel.
// It does not need to be handled explicitly.
type metadataClosedMsg struct{}

// listenForMetadataEvent returns a tea.Cmd that listens for metadata events
// without blocking the main Update loop.
func (m *Model) listenForMetadataEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.metadata
		if !ok {
			return metadataClosedMsg{}
		}
		return event
	}
}

// lyricsClosedMsg is returned when the lyrics event channel is closed.
// This prevents infinite loops in Bubble Tea's background goroutines
// where it would otherwise continuously read zero-values from a closed channel.
// It does not need to be handled explicitly.
type lyricsClosedMsg struct{}

// listenForLyricsEvent returns a tea.Cmd that listens for lyrics events
// without blocking the main Update loop.
func (m *Model) listenForLyricsEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.lyrics
		if !ok {
			return lyricsClosedMsg{}
		}
		return event
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
