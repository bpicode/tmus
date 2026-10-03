package tabs_test

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	keys := tabs.KeyMap{Next: key.NewBinding(key.WithKeys("right"))}
	styles := tabs.Styles{Border: lipgloss.NewStyle().Border(lipgloss.DoubleBorder())}
	tests := []struct {
		name   string
		opts   []tabs.Option
		next   string
		border string
		bold   bool
	}{
		{name: "defaults", next: "tab", border: "┌", bold: true},
		{name: "keys", opts: []tabs.Option{tabs.WithKeyMap(keys)}, next: "right", border: "┌", bold: true},
		{name: "styles", opts: []tabs.Option{tabs.WithStyles(styles)}, next: "tab", border: "╔"},
		{
			name: "combined", opts: []tabs.Option{tabs.WithKeyMap(keys), tabs.WithStyles(styles)},
			next: "right", border: "╔",
		},
		{
			name: "last option wins", opts: []tabs.Option{
				tabs.WithKeyMap(keys), tabs.WithKeyMap(tabs.DefaultKeyMap()),
				tabs.WithStyles(styles), tabs.WithStyles(tabs.DefaultStyles()),
			},
			next: "tab", border: "┌", bold: true,
		},
		{name: "empty key map", opts: []tabs.Option{tabs.WithKeyMap(tabs.KeyMap{})}, border: "┌", bold: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}}, tt.opts...)
			require.NoError(t, err)
			assert.False(t, m.Focused())
			m.SetSize(13, 3)
			assert.True(t, strings.HasPrefix(ansi.Strip(m.View()), tt.border), "header uses the configured border")
			assert.Equal(t, tt.bold, m.Styles.ActiveTab.GetBold(), "styles replace rather than merge defaults")
			m.Focus()
			for _, press := range []tea.KeyPressMsg{{Code: tea.KeyTab}, {Code: tea.KeyRight}} {
				updated, cmd := m.Update(press)
				if press.String() == tt.next {
					assert.Equal(t, "two", updated.ActiveID())
					assert.NotNil(t, cmd)
				} else {
					assert.Equal(t, "one", updated.ActiveID())
					assert.Nil(t, cmd)
				}
			}
		})
	}
}
