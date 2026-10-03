package view

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/bpicode/tmus/internal/ui/theme"
)

// Model2 is the experimental tabbed UI, developed alongside Model.
type Model2 struct {
	tabs   tabs.Model
	width  int
	styles styles
}

// NewModel2 creates a tabbed layout with placeholder content.
func NewModel2(th theme.Theme) (*Model2, error) {
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
	return &Model2{tabs: tabModel, styles: newStyles(th)}, nil
}

// Init returns the initial command for the experimental UI.
func (m *Model2) Init() tea.Cmd {
	return m.tabs.Init()
}

// Update handles resizing, tab navigation, and quitting.
func (m *Model2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.tabs.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.tabs, cmd = m.tabs.Update(msg)
	return m, cmd
}

// View renders the tab frame and the selected tab's placeholder content.
func (m *Model2) View() tea.View {
	content := "loading..."
	if m.width > 0 {
		active, _ := m.tabs.Active()
		content = m.tabs.Render(active.Label + " content will be added next.\n\n" +
			"Tab / Shift+Tab: switch tabs\nq / Ctrl+C: quit")
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "tmus · experimental tabs"
	view.ForegroundColor = m.styles.foreground
	view.BackgroundColor = m.styles.background
	return view
}
