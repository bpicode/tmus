package home

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/view/home/browser"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist"
)

type Model struct {
	browser  *browser.Model
	playlist *playlist.Model
	width    int
	height   int
	show     bool
}

func NewModel(cwd string, appRef *core.App) *Model {
	return &Model{
		browser:  browser.NewModel(cwd, appRef),
		playlist: playlist.NewModel(appRef),
	}
}

func (m *Model) UpdateSize(msg tea.WindowSizeMsg) {
	m.height = msg.Height
	m.width = msg.Width

	m.updateChildrenSize()
}

func (m *Model) updateChildrenSize() {
	leftWidth := max(m.width/2, 0)
	rightWidth := max(m.width-leftWidth, 0)

	browserWidth := leftWidth
	playlistWidth := rightWidth
	if !m.browser.Visible() {
		browserWidth = 0
		playlistWidth = m.width
	}
	m.browser.UpdateSize(tea.WindowSizeMsg{Width: browserWidth, Height: m.height})
	m.playlist.UpdateSize(tea.WindowSizeMsg{Width: playlistWidth, Height: m.height})
}

func (m *Model) HandleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.show {
		return nil, false
	}
	if cmd, handled := m.browser.HandleKey(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.playlist.HandleKey(msg); handled {
		return cmd, true
	}
	switch msg.String() {
	case "tab":
		m.browser.ToggleFocus()
		m.playlist.ToggleFocus()
		return nil, true
	case "b":
		m.browser.Toggle()
		if m.browser.Visible() {
			m.playlist.Focus(false)
			m.browser.Focus(true)
		} else {
			m.browser.Focus(false)
			m.playlist.Focus(true)
		}
		m.updateChildrenSize()
		return nil, true
	}
	return nil, false
}

func (m *Model) View() string {
	if !m.show {
		return ""
	}
	if m.width <= 0 || m.height <= 0 {
		return "loading..."
	}
	if m.browser.Visible() {
		left := m.browser.View()
		right := m.playlist.View()
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	return m.playlist.View()
}

func (m *Model) ShowBrowser(show bool) {
	m.browser.Show(show)
}

func (m *Model) FocusBrowser() {
	m.browser.Focus(true)
	m.playlist.Focus(false)
}

func (m *Model) FocusPlaylist() {
	m.browser.Focus(false)
	m.playlist.Focus(true)
}

func (m *Model) BrowserCwd() string {
	return m.browser.Cwd
}

func (m *Model) BrowserHidden() bool {
	return !m.browser.Visible()
}

func (m *Model) PlaylistFocused() bool {
	return m.playlist.Focused()
}

func (m *Model) Init() tea.Cmd {
	m.playlist.Show(true)
	return tea.Batch(
		m.browser.Init(),
		m.playlist.Init(),
	)
}

func (m *Model) Show(show bool) {
	m.show = show
}

func (m *Model) SyncState() {
	m.browser.SyncState()
	m.playlist.SyncState()
}
