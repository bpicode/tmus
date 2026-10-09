package help

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/bar"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type Model struct {
	show          bool
	lines         []string
	maxLineLength int
	width         int
	height        int
	viewport      viewport.Model
	styles        styles
}

func NewModel(th theme.Theme) *Model {
	styles := newStyles(th)
	lines := keybindings.render(styles)
	maxLineLength := lipgloss.Width(keybindings.appendix)
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
		styles:        styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) View() string {
	if !m.show || m.width <= m.styles.padding.GetHorizontalFrameSize() || m.height <= m.styles.padding.GetVerticalFrameSize() {
		return ""
	}
	width := m.viewport.Width()
	content := m.styles.footer.MaxWidth(width).Render(keybindings.appendix)
	if m.height-m.styles.padding.GetVerticalFrameSize() >= 2 {
		separator := bar.Horizontal(width, bar.WithStyle(m.styles.separator)).View()
		content = separator + "\n" + content
	}
	if m.viewport.Height() > 0 {
		content = m.viewport.View() + "\n" + content
	}
	styled := m.styles.padding.Render(content)
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
	m.viewport.SetWidth(max(min(m.maxLineLength, m.width-m.styles.padding.GetHorizontalFrameSize()), 0))
	// Reserve one row for the separator and one for the scroll hint.
	m.viewport.SetHeight(max(m.height-m.styles.padding.GetVerticalFrameSize()-2, 0))
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
