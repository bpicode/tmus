package view

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist"
)

// Model2 is the experimental tabbed UI, developed alongside Model.
type Model2 struct {
	app      *core.App
	playlist *playlist.Model
	tabs     tabs.Model
	events   eventChannels
	saved    State
	width    int
	styles   styles
}

// NewModel2 creates a tabbed player with a playlist and placeholder content for
// the remaining tabs. It restores the queue and opens supplied audio files.
func NewModel2(appRef *core.App, openFiles []string, cfg config.TUIConfig, th theme.Theme) (*Model2, error) {
	tabStyles := tabs.DefaultStyles()
	tabStyles.ActiveTab = lipgloss.NewStyle().Bold(true).Foreground(th.Primary)
	tabStyles.InactiveTab = lipgloss.NewStyle().Foreground(th.Muted)
	tabStyles.Border = tabStyles.Border.BorderForeground(th.Primary)
	tabModel, err := tabs.New([]tabs.Tab{
		{ID: "playlist", Label: "Playlist"},
		{ID: "track", Label: "Track"},
		{ID: "lyrics", Label: "Lyrics"},
		{ID: "browser", Label: "Browser"},
		{ID: "help", Label: "Help"},
	}, tabs.WithStyles(tabStyles))
	if err != nil {
		return nil, err
	}
	tabModel.Focus()
	saved, err := loadState()
	if err != nil {
		saved = State{}
	}
	m := &Model2{
		app: appRef, tabs: tabModel, saved: saved, styles: newStyles(th),
		playlist: playlist.NewModel(playlist.Config{Theme: th, App: appRef, FPS: cfg.FPS}),
	}
	m.restorePlayer()
	m.openFiles(openFiles)
	m.updatePlaylistFocus()
	return m, nil
}

// Init returns the initial command for the experimental UI.
func (m *Model2) Init() tea.Cmd {
	m.events.state, m.events.unsubState = m.app.SubscribeStateEvents()
	m.events.metadata, m.events.unsubMetadata = m.app.SubscribeMetadataEvents()
	return tea.Batch(m.tabs.Init(), m.playlist.Init(),
		m.listenForStateEvent(), m.listenForMetadataEvent(), tickCmd())
}

// Update routes keyboard input to the active tab and keeps the playlist updated
// with background messages regardless of the selected tab.
func (m *Model2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.tabs.SetSize(msg.Width, msg.Height)
		width, height := m.tabs.ContentSize()
		var cmd tea.Cmd
		m.playlist, cmd, _ = m.playlist.Update(tea.WindowSizeMsg{Width: width, Height: height})
		return m, cmd
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if key.Matches(msg, m.tabs.KeyMap.Next, m.tabs.KeyMap.Previous) {
			var cmd tea.Cmd
			m.tabs, cmd = m.tabs.Update(msg)
			m.updatePlaylistFocus()
			return m, cmd
		}
		if m.tabs.ActiveID() == "playlist" {
			var cmd tea.Cmd
			var stop bool
			m.playlist, cmd, stop = m.playlist.Update(msg)
			if stop {
				return m, cmd
			}
			cmds = append(cmds, cmd)
		}
		if msg.String() == "q" {
			return m, tea.Quit
		}
		return m, tea.Batch(cmds...)
	case core.StateEvent:
		cmds = append(cmds, m.listenForStateEvent())
	case core.MetadataEvent:
		cmds = append(cmds, m.listenForMetadataEvent())
	case tickMsg:
		cmds = append(cmds, tickCmd())
	case playlist.ToggleTrackInfoMsg:
		_ = m.tabs.Select("track")
		m.updatePlaylistFocus()
	case playlist.ToggleLyricsMsg:
		_ = m.tabs.Select("lyrics")
		m.updatePlaylistFocus()
	}
	var cmd tea.Cmd
	m.playlist, cmd, _ = m.playlist.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model2) updatePlaylistFocus() {
	active := m.tabs.ActiveID() == "playlist"
	m.playlist.Show(active)
	m.playlist.Focus(active)
}

// View renders the tab frame and the selected tab's content.
func (m *Model2) View() tea.View {
	content := "loading..."
	if m.width > 0 {
		active, _ := m.tabs.Active()
		content = active.Label + " content will be added next.\n\n" +
			"Tab / Shift+Tab: switch tabs\nq / Ctrl+C: quit"
		if active.ID == "playlist" {
			content = m.playlist.View()
		}
		content = m.tabs.Render(content)
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "tmus · experimental tabs"
	view.ForegroundColor = m.styles.foreground
	view.BackgroundColor = m.styles.background
	return view
}

// Shutdown releases resources owned by the UI. The runner owns app shutdown.
func (m *Model2) Shutdown() {
	if m.events.unsubState != nil {
		m.events.unsubState()
		m.events.unsubState = nil
	}
	if m.events.unsubMetadata != nil {
		m.events.unsubMetadata()
		m.events.unsubMetadata = nil
	}
}

func (m *Model2) listenForStateEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.state
		if !ok {
			return stateClosedMsg{}
		}
		return event
	}
}

func (m *Model2) listenForMetadataEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.metadata
		if !ok {
			return metadataClosedMsg{}
		}
		return event
	}
}
