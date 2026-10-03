package tabs

import tea "charm.land/bubbletea/v2"

// Model holds the state of a tab bar.
type Model struct{}

// New creates a tab bar model.
func New() Model {
	return Model{}
}

// Init returns the initial command for the component.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles a Bubble Tea message.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

// View renders the tab bar.
func (m Model) View() string {
	return ""
}
