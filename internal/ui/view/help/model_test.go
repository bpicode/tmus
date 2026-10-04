package help

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestModelKeepsFooterVisible(t *testing.T) {
	m := NewModel(theme.Theme{})
	m.Show(true)
	for _, size := range []tea.WindowSizeMsg{
		{Width: 80, Height: 24},
		{Width: 60, Height: 12},
		{Width: 40, Height: 6},
		{Width: 40, Height: 3},
	} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m.Update(size)
			for _, key := range []rune{tea.KeyHome, tea.KeyDown, tea.KeyEnd, tea.KeyHome} {
				m.Update(tea.KeyPressMsg{Code: key})
				content := ansi.Strip(m.View())
				require.Equal(t, size.Height, lipgloss.Height(content))
				assert.Equal(t, size.Width, lipgloss.Width(content))
				lines := strings.Split(content, "\n")
				assert.Equal(t, keybindings.appendix, strings.TrimSpace(lines[len(lines)-2]))
				if size.Height > 3 {
					assert.Equal(t, strings.Repeat("─", m.viewport.Width()), strings.TrimSpace(lines[len(lines)-3]))
				}
			}
		})
	}
}
