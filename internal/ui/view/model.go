package view

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/components/notification"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view/browser"
	"github.com/bpicode/tmus/internal/ui/view/help"
	"github.com/bpicode/tmus/internal/ui/view/lyrics"
	"github.com/bpicode/tmus/internal/ui/view/playlist"
	"github.com/bpicode/tmus/internal/ui/view/track_info"
)

// Model is the tabbed player UI.
type Model struct {
	notifications *notification.Model
	app           *core.App
	playlist      *playlist.Model
	trackInfo     *track_info.Model
	lyrics        *lyrics.Model
	browser       *browser.Model
	help          *help.Model
	tabs          *tabs.Model
	events        eventChannels
	width         int
	styles        styles
}

// NewModel creates the tabbed player. It restores
// saved state, uses startDir when supplied, and opens supplied audio files.
func NewModel(appRef *core.App, startDir string, openFiles []string, cfg config.TUIConfig, th theme.Theme) (*Model, error) {
	tabStyles := tabs.DefaultStyles()
	tabStyles.ActiveTab = lipgloss.NewStyle().Bold(true).Foreground(th.Primary)
	tabStyles.InactiveTab = lipgloss.NewStyle().Foreground(th.Muted)
	tabStyles.Border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary)
	tabModel, err := tabs.New([]tabs.Tab{
		{ID: "playlist", Label: "🎧 Playlist"},
		{ID: "track", Label: "🎵 Track"},
		{ID: "lyrics", Label: "📜 Lyrics"},
		{ID: "browser", Label: "📂 Browser"},
		{ID: "help", Label: "❓ Help"},
	}, tabs.WithStyles(tabStyles))
	if err != nil {
		return nil, err
	}
	tabModel.Focus()
	saved, err := loadState()
	if err != nil {
		saved = State{}
	}
	m := &Model{
		app: appRef, tabs: tabModel, styles: newStyles(th),
		notifications: notification.New(newNotificationStyles(th)),
		playlist:      playlist.NewModel(playlist.Config{Theme: th, App: appRef, FPS: cfg.FPS}),
		trackInfo: track_info.NewModel(track_info.Config{
			Theme: th, App: appRef, ArtworkAspect: cfg.ArtworkAspect, ArtworkRenderer: cfg.ArtworkRenderer,
		}),
		lyrics: lyrics.NewModel(lyrics.Config{Theme: th, App: appRef, FollowLine: saved.Lyrics.FollowLine}),
		browser: browser.NewModel(browser.Config{
			Cwd:     initialBrowserDir(appRef.Library(), startDir, saved.Browser.Cwd),
			HomeDir: cfg.BrowserHome, Theme: th, App: appRef, Library: appRef.Library(),
		}),
		help: help.NewModel(th),
	}
	m.restorePlayer(saved.Player)
	m.openFiles(openFiles)
	m.playlist.Show(true)
	m.playlist.Focus(true)
	return m, nil
}

// Init subscribes to app events and initializes the tab content.
func (m *Model) Init() tea.Cmd {
	m.events.state, m.events.unsubState = m.app.SubscribeStateEvents()
	m.events.metadata, m.events.unsubMetadata = m.app.SubscribeMetadataEvents()
	m.events.lyrics, m.events.unsubLyrics = m.app.SubscribeLyricsEvents()
	return tea.Batch(m.tabs.Init(), playlistCmd(m.playlist.Init()), m.trackInfo.Init(), m.lyrics.Init(),
		browserCmd(m.browser.Init()), m.help.Init(),
		m.listenForStateEvent(), m.listenForMetadataEvent(), m.listenForLyricsEvent(), tickCmd())
}

// Update routes keyboard input to the active tab and background messages to
// the content models.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.notifications.Update(msg); handled {
		return m, cmd
	}
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case browserMsg:
		if cmd, handled := m.notifications.Update(msg.msg); handled {
			return m, cmd
		}
		if batch, ok := msg.msg.(tea.BatchMsg); ok {
			return m, childBatch(batch, browserCmd)
		}
		var cmd tea.Cmd
		m.browser, cmd, _ = m.browser.Update(msg.msg)
		return m, browserCmd(cmd)
	case playlistMsg:
		if cmd, handled := m.notifications.Update(msg.msg); handled {
			return m, cmd
		}
		switch reply := msg.msg.(type) {
		case tea.BatchMsg:
			return m, childBatch(reply, playlistCmd)
		case playlist.ToggleTrackInfoMsg, playlist.ToggleLyricsMsg:
			return m.Update(reply)
		}
		var cmd tea.Cmd
		m.playlist, cmd, _ = m.playlist.Update(msg.msg)
		return m, playlistCmd(cmd)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.tabs.SetSize(msg.Width, msg.Height)
		width, height := m.tabs.ContentSize()
		size := tea.WindowSizeMsg{Width: width, Height: height}
		var cmd tea.Cmd
		m.playlist, cmd, _ = m.playlist.Update(size)
		cmds = append(cmds, playlistCmd(cmd))
		m.trackInfo, cmd, _ = m.trackInfo.Update(size)
		cmds = append(cmds, cmd)
		m.lyrics, cmd, _ = m.lyrics.Update(size)
		cmds = append(cmds, cmd)
		m.browser, cmd, _ = m.browser.Update(size)
		cmds = append(cmds, browserCmd(cmd))
		m.help, cmd, _ = m.help.Update(size)
		return m, tea.Batch(append(cmds, cmd)...)
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.lyrics.Shutdown()
			return m, tea.Sequence(m.trackInfo.Show(false), tea.Quit)
		}
		if key.Matches(msg, m.tabs.KeyMap.Next, m.tabs.KeyMap.Previous) {
			var cmd tea.Cmd
			m.tabs, cmd = m.tabs.Update(msg)
			return m, tea.Batch(cmd, m.updateActiveTab())
		}
		if m.tabs.ActiveID() == "playlist" {
			var cmd tea.Cmd
			var stop bool
			m.playlist, cmd, stop = m.playlist.Update(msg)
			cmd = playlistCmd(cmd)
			if stop {
				return m, cmd
			}
			cmds = append(cmds, cmd)
		}
		if m.tabs.ActiveID() == "browser" {
			var cmd tea.Cmd
			var stop bool
			m.browser, cmd, stop = m.browser.Update(msg)
			cmd = browserCmd(cmd)
			if stop {
				return m, cmd
			}
			cmds = append(cmds, cmd)
		}
		if msg.String() == "esc" && m.tabs.ActiveID() == "browser" {
			_ = m.tabs.Select("playlist")
			return m, m.updateActiveTab()
		}
		if msg.String() == "b" {
			switch m.tabs.ActiveID() {
			case "playlist":
				_ = m.tabs.Select("browser")
				return m, m.updateActiveTab()
			case "browser":
				_ = m.tabs.Select("playlist")
				return m, m.updateActiveTab()
			}
		}
		if msg.String() == "q" {
			m.lyrics.Shutdown()
			return m, tea.Sequence(m.trackInfo.Show(false), tea.Quit)
		}
		if msg.String() == "?" {
			tab := "help"
			if m.tabs.ActiveID() == "help" {
				tab = "playlist"
			}
			_ = m.tabs.Select(tab)
			return m, m.updateActiveTab()
		}
		if m.tabs.ActiveID() == "track" {
			if msg.String() == "esc" || msg.String() == "i" {
				_ = m.tabs.Select("playlist")
				return m, m.updateActiveTab()
			}
			var cmd tea.Cmd
			m.trackInfo, cmd, _ = m.trackInfo.Update(msg)
			cmds = append(cmds, cmd)
		}
		if m.tabs.ActiveID() == "lyrics" {
			if msg.String() == "esc" || msg.String() == "L" {
				_ = m.tabs.Select("playlist")
				return m, m.updateActiveTab()
			}
			var cmd tea.Cmd
			m.lyrics, cmd, _ = m.lyrics.Update(msg)
			cmds = append(cmds, cmd)
		}
		if m.tabs.ActiveID() == "help" {
			if msg.String() == "esc" {
				_ = m.tabs.Select("playlist")
				return m, m.updateActiveTab()
			}
			var cmd tea.Cmd
			m.help, cmd, _ = m.help.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case core.StateEvent:
		if msg.Source == core.StateEventCommand && msg.Changes&core.StateChangePlaylist != 0 {
			switch msg.Command.Type {
			case core.CmdAdd:
				cmds = append(cmds, notification.Success("File added"))
			case core.CmdAddAll:
				cmds = append(cmds, notification.Success(addedFilesText(len(msg.Command.Tracks))))
			}
		}
		cmds = append(cmds, m.listenForStateEvent())
		if m.tabs.ActiveID() == "track" {
			cmds = append(cmds, m.trackInfo.Show(true))
		}
		if m.tabs.ActiveID() == "lyrics" {
			hasTracks := len(m.app.State().Playlist) > 0
			if !hasTracks || !m.lyrics.Visible() {
				m.lyrics.Show(hasTracks)
			}
		}
	case core.MetadataEvent:
		cmds = append(cmds, m.listenForMetadataEvent())
	case core.LyricsEvent:
		cmds = append(cmds, m.listenForLyricsEvent())
	case tickMsg:
		cmds = append(cmds, tickCmd())
	case playlist.ToggleTrackInfoMsg:
		_ = m.tabs.Select("track")
		cmds = append(cmds, m.updateActiveTab())
	case playlist.ToggleLyricsMsg:
		_ = m.tabs.Select("lyrics")
		cmds = append(cmds, m.updateActiveTab())
	}
	var cmd tea.Cmd
	m.playlist, cmd, _ = m.playlist.Update(msg)
	cmds = append(cmds, playlistCmd(cmd))
	m.trackInfo, cmd, _ = m.trackInfo.Update(msg)
	cmds = append(cmds, cmd)
	m.lyrics, cmd, _ = m.lyrics.Update(msg)
	cmds = append(cmds, cmd)
	m.browser, cmd, _ = m.browser.Update(msg)
	cmds = append(cmds, browserCmd(cmd))
	m.help, cmd, _ = m.help.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) updateActiveTab() tea.Cmd {
	active := m.tabs.ActiveID() == "playlist"
	m.playlist.Show(active)
	m.playlist.Focus(active)
	active = m.tabs.ActiveID() == "browser"
	m.browser.Show(active)
	m.browser.Focus(active)
	m.lyrics.Show(m.tabs.ActiveID() == "lyrics")
	m.help.Show(m.tabs.ActiveID() == "help")
	return m.trackInfo.Show(m.tabs.ActiveID() == "track")
}

// View renders the tab frame and the selected tab's content.
func (m *Model) View() tea.View {
	content := "loading..."
	if m.width > 0 {
		active, _ := m.tabs.Active()
		content = ""
		switch active.ID {
		case "playlist":
			content = m.playlist.View()
		case "track":
			content = "No track selected."
			if m.trackInfo.Visible() {
				content = m.trackInfo.View()
			}
		case "lyrics":
			content = "No track selected."
			if m.lyrics.Visible() {
				content = m.lyrics.View()
			}
		case "browser":
			content = m.browser.View()
		case "help":
			content = m.help.View()
		}
		content = m.tabs.Render(content)
	}
	view := tea.NewView(m.notifications.Overlay(content))
	view.AltScreen = true
	view.WindowTitle = "tmus"
	view.ForegroundColor = m.styles.foreground
	view.BackgroundColor = m.styles.background
	return view
}

// Shutdown releases resources owned by the UI. The runner owns app shutdown.
func (m *Model) Shutdown() {
	m.lyrics.Shutdown()
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
}

func (m *Model) listenForStateEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.state
		if !ok {
			return stateClosedMsg{}
		}
		return event
	}
}

func (m *Model) listenForMetadataEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.metadata
		if !ok {
			return metadataClosedMsg{}
		}
		return event
	}
}

func (m *Model) listenForLyricsEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.lyrics
		if !ok {
			return lyricsClosedMsg{}
		}
		return event
	}
}

func addedFilesText(count int) string {
	if count == 1 {
		return "File added"
	}
	return fmt.Sprintf("%d files added", count)
}
