package tabs

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Tab describes a tab independently of its content.
type Tab struct {
	// ID is a non-empty, unique identifier, independent of the display label.
	ID string
	// Label is the display text. Labels may be empty or shared by multiple tabs.
	Label string
}

// ChangeMsg reports a selection change caused by keyboard navigation.
// Previous and Current are tab IDs. Programmatic selection does not emit it.
type ChangeMsg struct {
	Previous string
	Current  string
}

// Model holds the state of a tab bar. Its zero value has no tabs or selection.
type Model struct {
	// KeyMap configures the navigation keys recognized by Update.
	KeyMap KeyMap

	tabs    []Tab
	active  int
	focused bool
}

// New creates a tab bar model with the first tab selected, or no selection if
// items is empty. It copies items and rejects empty or duplicate IDs.
// The model starts unfocused, with DefaultKeyMap bindings.
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
	return Model{tabs: slices.Clone(items), KeyMap: DefaultKeyMap()}, nil
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

// Next selects the next tab, wrapping to the first tab after the last.
// It works regardless of focus and is a no-op for empty and single-tab models.
func (m *Model) Next() {
	if len(m.tabs) < 2 {
		return
	}
	m.active = (m.active + 1) % len(m.tabs)
}

// Previous selects the previous tab, wrapping to the last tab before the first.
// It works regardless of focus and is a no-op for empty and single-tab models.
func (m *Model) Previous() {
	if len(m.tabs) < 2 {
		return
	}
	m.active = (m.active + len(m.tabs) - 1) % len(m.tabs)
}

// Focus enables keyboard navigation.
func (m *Model) Focus() {
	m.focused = true
}

// Blur disables keyboard navigation without changing the selection.
func (m *Model) Blur() {
	m.focused = false
}

// Focused reports whether keyboard navigation is enabled.
func (m Model) Focused() bool {
	return m.focused
}

// Init returns the initial command for the component.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles navigation key presses while focused. When the selection
// changes, it returns a command that emits a ChangeMsg with the transition.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok || !m.focused {
		return m, nil
	}
	previous := m.ActiveID()
	switch {
	case key.Matches(press, m.KeyMap.Next):
		m.Next()
	case key.Matches(press, m.KeyMap.Previous):
		m.Previous()
	}
	if previous == m.ActiveID() {
		return m, nil
	}
	change := ChangeMsg{Previous: previous, Current: m.ActiveID()}
	return m, func() tea.Msg { return change }
}

// View renders the tab bar.
func (m Model) View() string {
	return ""
}
