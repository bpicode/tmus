package view_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel2Navigation(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
		id   string
		text string
	}{
		{name: "next", key: tea.KeyPressMsg{Code: tea.KeyTab}, id: "track", text: "Track"},
		{name: "previous wraps", key: tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, id: "help", text: "Help"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := view.NewModel2(theme.Resolve(config.Default().TUI.Theme))
			require.NoError(t, err)
			_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			assert.Contains(t, m.View().Content, "Playlist content will be added next.")
			updated, cmd := m.Update(tt.key)
			require.NotNil(t, cmd)
			change := cmd()
			assert.Equal(t, tabs.ChangeMsg{Previous: "playlist", Current: tt.id}, change)
			assert.Contains(t, updated.View().Content, tt.text+" content will be added next.")
			_, cmd = m.Update(change)
			assert.Nil(t, cmd)
			assert.Contains(t, m.View().Content, tt.text+" content will be added next.")
		})
	}
}

func TestModel2ResizesFrame(t *testing.T) {
	th := theme.Resolve(config.Default().TUI.Theme)
	m, err := view.NewModel2(th)
	require.NoError(t, err)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 60, Height: 12}} {
		_, cmd := m.Update(size)
		assert.Nil(t, cmd)
		v := m.View()
		assert.Equal(t, size.Width, lipgloss.Width(v.Content))
		assert.Equal(t, size.Height, lipgloss.Height(v.Content))
		assert.Contains(t, ansi.Strip(v.Content), "│ Playlist │ Track │ Lyrics │ Browser │ Help │")
		assert.True(t, v.AltScreen)
		assert.Equal(t, th.Foreground, v.ForegroundColor)
		assert.Equal(t, th.Background, v.BackgroundColor)
	}
}

func TestModel2Quits(t *testing.T) {
	for _, press := range []tea.KeyPressMsg{{Code: 'q'}, {Code: 'c', Mod: tea.ModCtrl}} {
		t.Run(press.String(), func(t *testing.T) {
			m, err := view.NewModel2(theme.Resolve(config.Default().TUI.Theme))
			require.NoError(t, err)
			_, cmd := m.Update(press)
			require.NotNil(t, cmd)
			assert.IsType(t, tea.QuitMsg{}, cmd())
		})
	}
}
