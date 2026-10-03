package tabs_test

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigation(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		initial string
		move    func(*tabs.Model)
		want    string
	}{
		{"next", 3, "one", (*tabs.Model).Next, "two"},
		{"next wraps", 3, "three", (*tabs.Model).Next, "one"},
		{"previous", 3, "three", (*tabs.Model).Previous, "two"},
		{"previous wraps", 3, "one", (*tabs.Model).Previous, "three"},
		{"empty next", 0, "", (*tabs.Model).Next, ""},
		{"empty previous", 0, "", (*tabs.Model).Previous, ""},
		{"single next", 1, "one", (*tabs.Model).Next, "one"},
		{"single previous", 1, "one", (*tabs.Model).Previous, "one"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []tabs.Tab{{ID: "one"}, {ID: "two"}, {ID: "three"}}
			m, err := tabs.New(items[:tt.count])
			require.NoError(t, err)
			if tt.initial != "" {
				require.NoError(t, m.Select(tt.initial))
			}
			tt.move(&m)
			assert.Equal(t, tt.want, m.ActiveID())
			assert.False(t, m.Focused(), "programmatic navigation does not require focus")
		})
	}
}

func TestFocus(t *testing.T) {
	m, err := tabs.New([]tabs.Tab{{ID: "one"}, {ID: "two"}})
	require.NoError(t, err)
	assert.False(t, m.Focused())
	m.Focus()
	assert.True(t, m.Focused())
	require.NoError(t, m.Select("two"))
	m.Blur()
	assert.False(t, m.Focused())
	assert.Equal(t, "two", m.ActiveID())
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, "two", updated.ActiveID())
	assert.Nil(t, cmd)
	m.Focus()
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Equal(t, "one", updated.ActiveID())
	require.NotNil(t, cmd)
}

func TestUpdateNavigation(t *testing.T) {
	tab := tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	custom := tabs.KeyMap{
		Next:     key.NewBinding(key.WithKeys("right")),
		Previous: key.NewBinding(key.WithKeys("left")),
	}
	disabled := tabs.DefaultKeyMap()
	disabled.Next.SetEnabled(false)
	disabled.Previous.SetEnabled(false)
	unbound := tabs.KeyMap{}
	tests := []struct {
		name    string
		initial string
		focused bool
		keys    *tabs.KeyMap
		msg     tea.Msg
		want    string
	}{
		{name: "next", initial: "one", focused: true, msg: tab, want: "two"},
		{name: "next wraps", initial: "three", focused: true, msg: tab, want: "one"},
		{name: "previous", initial: "three", focused: true, msg: shiftTab, want: "two"},
		{name: "previous wraps", initial: "one", focused: true, msg: shiftTab, want: "three"},
		{name: "unfocused next", initial: "one", msg: tab, want: "one"},
		{name: "unfocused previous", initial: "one", msg: shiftTab, want: "one"},
		{name: "unrelated key", initial: "one", focused: true, msg: tea.KeyPressMsg{Code: tea.KeyDown}, want: "one"},
		{name: "key release", initial: "one", focused: true, msg: tea.KeyReleaseMsg{Code: tea.KeyTab}, want: "one"},
		{name: "window size", initial: "one", focused: true, msg: tea.WindowSizeMsg{Width: 80, Height: 24}, want: "one"},
		{name: "change message", initial: "one", focused: true, msg: tabs.ChangeMsg{Previous: "one", Current: "two"}, want: "one"},
		{name: "custom next", initial: "one", focused: true, keys: &custom, msg: tea.KeyPressMsg{Code: tea.KeyRight}, want: "two"},
		{name: "custom previous", initial: "one", focused: true, keys: &custom, msg: tea.KeyPressMsg{Code: tea.KeyLeft}, want: "three"},
		{name: "replaced default", initial: "one", focused: true, keys: &custom, msg: tab, want: "one"},
		{name: "disabled next", initial: "one", focused: true, keys: &disabled, msg: tab, want: "one"},
		{name: "disabled previous", initial: "one", focused: true, keys: &disabled, msg: shiftTab, want: "one"},
		{name: "unbound keys", initial: "one", focused: true, keys: &unbound, msg: tab, want: "one"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New([]tabs.Tab{{ID: "one"}, {ID: "two"}, {ID: "three"}})
			require.NoError(t, err)
			require.NoError(t, m.Select(tt.initial))
			if tt.focused {
				m.Focus()
			}
			if tt.keys != nil {
				m.KeyMap = *tt.keys
			}
			updated, cmd := m.Update(tt.msg)
			assert.Equal(t, tt.want, updated.ActiveID())
			assert.Equal(t, tt.focused, updated.Focused())
			assert.Equal(t, tt.initial, m.ActiveID(), "Update must not mutate the input model")
			if tt.initial == tt.want {
				assert.Nil(t, cmd)
				return
			}
			require.NotNil(t, cmd)
			// Later selection must not alter an already queued notification.
			require.NoError(t, updated.Select(tt.initial))
			assert.Equal(t, tabs.ChangeMsg{Previous: tt.initial, Current: tt.want}, cmd())
		})
	}
}

func TestUpdateWithoutNavigationTargets(t *testing.T) {
	for _, count := range []int{0, 1} {
		for _, msg := range []tea.KeyPressMsg{
			{Code: tea.KeyTab},
			{Code: tea.KeyTab, Mod: tea.ModShift},
		} {
			t.Run(fmt.Sprintf("%d tabs/%s", count, msg.String()), func(t *testing.T) {
				items := []tabs.Tab{{ID: "one"}}
				m, err := tabs.New(items[:count])
				require.NoError(t, err)
				m.Focus()
				updated, cmd := m.Update(msg)
				assert.Equal(t, m.ActiveID(), updated.ActiveID())
				assert.Nil(t, cmd)
			})
		}
	}
	var zero tabs.Model
	zero.Focus()
	updated, cmd := zero.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Empty(t, updated.ActiveID())
	assert.Nil(t, cmd)
}
