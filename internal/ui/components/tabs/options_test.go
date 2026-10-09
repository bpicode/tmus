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
	keys := tabs.KeyMap{
		Next:     key.NewBinding(key.WithKeys("right")),
		Previous: key.NewBinding(key.WithKeys("left")),
	}
	styles := tabs.Styles{Border: lipgloss.NewStyle().Border(lipgloss.DoubleBorder())}
	tests := []struct {
		name     string
		opts     []tabs.Option
		next     string
		previous string
		border   string
		bold     bool
	}{
		{name: "defaults", next: "tab", previous: "shift+tab", border: "┌", bold: true},
		{name: "keys", opts: []tabs.Option{tabs.WithKeyMap(keys)}, next: "right", previous: "left", border: "┌", bold: true},
		{name: "styles", opts: []tabs.Option{tabs.WithStyles(styles)}, next: "tab", previous: "shift+tab", border: "╔"},
		{
			name: "combined", opts: []tabs.Option{tabs.WithKeyMap(keys), tabs.WithStyles(styles)},
			next: "right", previous: "left", border: "╔",
		},
		{
			name: "last option wins", opts: []tabs.Option{
				tabs.WithKeyMap(keys), tabs.WithKeyMap(tabs.DefaultKeyMap()),
				tabs.WithStyles(styles), tabs.WithStyles(tabs.DefaultStyles()),
			},
			next: "tab", previous: "shift+tab", border: "┌", bold: true,
		},
		{name: "empty key map", opts: []tabs.Option{tabs.WithKeyMap(tabs.KeyMap{})}, border: "┌", bold: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}, {ID: "three", Label: "Three"}}, tt.opts...)
			require.NoError(t, err)
			assert.False(t, m.Focused())
			m.SetSize(21, 3)
			assert.True(t, strings.HasPrefix(ansi.Strip(m.View()), tt.border), "header uses the configured border")
			assert.Equal(t, tt.bold, m.Styles.ActiveTab.GetBold(), "styles replace rather than merge defaults")
			m.Focus()
			for _, press := range []tea.KeyPressMsg{
				{Code: tea.KeyTab}, {Code: tea.KeyTab, Mod: tea.ModShift},
				{Code: tea.KeyRight}, {Code: tea.KeyLeft},
			} {
				require.NoError(t, m.Select("one"))
				want := "one"
				switch press.String() {
				case tt.next:
					want = "two"
				case tt.previous:
					want = "three"
				}
				updated, cmd := m.Update(press)
				assert.Equal(t, want, updated.ActiveID(), "key %s", press.String())
				if want != "one" {
					assert.NotNil(t, cmd)
				} else {
					assert.Nil(t, cmd)
				}
			}
		})
	}
}
