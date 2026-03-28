package home

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
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

func NewModel(cwd string, cfg config.TUIConfig, appRef *core.App) *Model {
	return &Model{
		browser:  browser.NewModel(cwd, cfg, appRef),
		playlist: playlist.NewModel(appRef),
	}
}

func (m *Model) Init() tea.Cmd {
	m.playlist.Show(true)
	return tea.Batch(
		m.browser.Init(),
		m.playlist.Init(),
	)
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleSizeMsg(msg)
	case tea.KeyPressMsg:
		return m.handleKeyPressMsg(msg)
	default:
		return m.handleRemaining(msg)
	}
}

func (m *Model) handleSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd, bool) {
	m.height = msg.Height
	m.width = msg.Width

	var cmds []tea.Cmd
	var cmdSub tea.Cmd
	browserSize, playlistSize := m.childrenSizes()
	m.browser, cmdSub, _ = m.browser.Update(browserSize)
	cmds = append(cmds, cmdSub)
	m.playlist, cmdSub, _ = m.playlist.Update(playlistSize)
	cmds = append(cmds, cmdSub)
	return m, tea.Batch(cmds...), false
}

func (m *Model) handleKeyPressMsg(msg tea.KeyPressMsg) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}
	switch msg.String() {
	case "tab":
		if m.browser.Visible() {
			m.browser.ToggleFocus()
			m.playlist.ToggleFocus()
		}
		return m, nil, true
	case "b":
		if m.browser.Searching() || m.playlist.Searching() {
			return m.handleRemaining(msg)
		}
		m.browser.Toggle()
		if m.browser.Visible() {
			m.playlist.Focus(false)
			m.browser.Focus(true)
		} else {
			m.browser.Focus(false)
			m.playlist.Focus(true)
		}
		var cmds []tea.Cmd
		var cmdSub tea.Cmd
		browserSize, playlistSize := m.childrenSizes()
		m.browser, cmdSub, _ = m.browser.Update(browserSize)
		cmds = append(cmds, cmdSub)
		m.playlist, cmdSub, _ = m.playlist.Update(playlistSize)
		cmds = append(cmds, cmdSub)
		return m, tea.Batch(cmds...), true
	default:
		return m.handleRemaining(msg)
	}
}

func (m *Model) handleRemaining(msg tea.Msg) (*Model, tea.Cmd, bool) {
	var cmds []tea.Cmd
	var cmdSub tea.Cmd
	var stop bool

	m.browser, cmdSub, stop = m.browser.Update(msg)
	if stop {
		return m, cmdSub, true
	}
	cmds = append(cmds, cmdSub)

	m.playlist, cmdSub, stop = m.playlist.Update(msg)
	if stop {
		return m, cmdSub, true
	}
	cmds = append(cmds, cmdSub)

	return m, tea.Batch(cmds...), false
}

func (m *Model) childrenSizes() (tea.WindowSizeMsg, tea.WindowSizeMsg) {
	leftWidth := max(m.width/2, 0)
	rightWidth := max(m.width-leftWidth, 0)

	browserWidth := leftWidth
	playlistWidth := rightWidth
	if !m.browser.Visible() {
		browserWidth = 0
		playlistWidth = m.width
	}
	return tea.WindowSizeMsg{Width: browserWidth, Height: m.height}, tea.WindowSizeMsg{Width: playlistWidth, Height: m.height}
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

func (m *Model) Show(show bool) {
	m.show = show
}
