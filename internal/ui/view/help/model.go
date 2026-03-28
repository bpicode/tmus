package help

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	show          bool
	lines         []string
	maxLineLength int
	width         int
	height        int
	viewport      viewport.Model
}

func NewModel() *Model {
	lines := keybindings.render()
	maxLineLength := 0
	for _, line := range lines {
		maxLineLength = max(maxLineLength, lipgloss.Width(line))
	}
	vp := viewport.New()
	vp.LeftGutterFunc = viewport.NoGutter
	vp.SetContentLines(lines)
	return &Model{
		show:          false,
		lines:         lines,
		maxLineLength: maxLineLength,
		viewport:      vp,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) View() string {
	if !m.show {
		return ""
	}
	styled := styleOverlay.Render(m.viewport.View())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, styled)
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleSizeMsg(msg)
	case tea.KeyPressMsg:
		return m.handleKeyPressMsg(msg)
	default:
		return m, nil, false
	}
}

func (m *Model) handleSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd, bool) {
	m.width = msg.Width
	m.height = msg.Height
	m.viewport.SetWidth(max(min(m.maxLineLength, m.width-styleOverlay.GetHorizontalFrameSize()), 0))
	m.viewport.SetHeight(max(m.height-styleOverlay.GetVerticalFrameSize(), 0))
	return m, nil, false
}

func (m *Model) handleKeyPressMsg(msg tea.KeyPressMsg) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}
	switch msg.String() {
	case "q", "esc", "?":
		m.Show(false)
		return m, nil, true
	case "up", "k":
		m.viewport.ScrollUp(1)
		return m, nil, true
	case "down", "j":
		m.viewport.ScrollDown(1)
		return m, nil, true
	case "pgup", "pageup":
		m.viewport.PageUp()
		return m, nil, true
	case "pgdown", "pagedown":
		m.viewport.PageDown()
		return m, nil, true
	case "home", "pos1":
		m.viewport.GotoTop()
		return m, nil, true
	case "end":
		m.viewport.GotoBottom()
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m *Model) Visible() bool {
	return m.show
}

func (m *Model) Show(show bool) {
	m.show = show
	if !m.show {
		m.viewport.GotoTop()
	}
}
