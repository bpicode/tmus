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

func (m *Model) Visible() bool {
	return m.show
}

func (m *Model) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.show {
		return nil, false
	}
	return nil, m.Update(msg)
}

func (m *Model) UpdateSize(message tea.WindowSizeMsg) {
	m.width = message.Width
	m.height = message.Height
	m.viewport.SetWidth(max(min(m.maxLineLength, m.width-styleOverlay.GetHorizontalFrameSize()), 0))
	m.viewport.SetHeight(max(m.height-styleOverlay.GetVerticalFrameSize(), 0))
}

func (m *Model) Show(show bool) {
	m.show = show
	if !m.show {
		m.viewport.GotoTop()
	}
}

func (m *Model) Update(msg tea.KeyMsg) bool {
	if !m.show {
		return false
	}

	switch msg.String() {
	case "q", "esc", "?":
		m.Show(false)
		return true
	case "up", "k":
		m.viewport.ScrollUp(1)
		return true
	case "down", "j":
		m.viewport.ScrollDown(1)
		return true
	case "pgup", "pageup":
		m.viewport.PageUp()
		return true
	case "pgdown", "pagedown":
		m.viewport.PageDown()
		return true
	case "home", "pos1":
		m.viewport.GotoTop()
		return true
	case "end":
		m.viewport.GotoBottom()
		return true
	default:
		return false
	}
}

func (m *Model) View() string {
	if !m.show {
		return ""
	}
	styled := styleOverlay.Render(m.viewport.View())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, styled)
}

func (m *Model) Init() tea.Cmd {
	return nil
}
