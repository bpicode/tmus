package help

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestModelShowsKeybindings(t *testing.T) {
	m := NewModel(theme.Theme{})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 100})
	m.Show(true)
	r := m.View()
	assert.Contains(t, strings.ToLower(r), "keybindings")
}

func TestModelRendersEmptyIfNotShown(t *testing.T) {
	m := NewModel(theme.Theme{})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 100})
	m.Show(false)
	r := m.View()
	assert.Empty(t, r)
}
