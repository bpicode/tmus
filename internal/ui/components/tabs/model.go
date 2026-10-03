package tabs

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
)

// Tab describes a tab independently of its content.
type Tab struct {
	// ID is a non-empty, unique identifier, independent of the display label.
	ID string
	// Label is the display text. Labels may be empty or shared by multiple tabs.
	Label string
}

// Model holds the state of a tab bar. Its zero value has no tabs or selection.
type Model struct {
	tabs   []Tab
	active int
}

// New creates a tab bar model with the first tab selected, or no selection if
// items is empty. It copies items and rejects empty or duplicate IDs.
func New(items []Tab) (Model, error) {
	ids := make(map[string]struct{}, len(items))
	for i, tab := range items {
		if tab.ID == "" {
			return Model{}, fmt.Errorf("tabs: tab at index %d has an empty ID", i)
		}
		if _, exists := ids[tab.ID]; exists {
			return Model{}, fmt.Errorf("tabs: duplicate tab ID %q", tab.ID)
		}
		ids[tab.ID] = struct{}{}
	}
	return Model{tabs: slices.Clone(items)}, nil
}

// Active returns the selected tab, or false if the model has no tabs.
func (m Model) Active() (Tab, bool) {
	if len(m.tabs) == 0 {
		return Tab{}, false
	}
	return m.tabs[m.active], true
}

// ActiveID returns the selected tab's ID, or an empty string if there is none.
func (m Model) ActiveID() string {
	tab, _ := m.Active()
	return tab.ID
}

// Select selects the tab with id. An unknown ID returns an error and leaves
// the current selection unchanged. Selecting the active tab is a no-op.
func (m *Model) Select(id string) error {
	for i, tab := range m.tabs {
		if tab.ID == id {
			m.active = i
			return nil
		}
	}
	return fmt.Errorf("tabs: unknown tab ID %q", id)
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
