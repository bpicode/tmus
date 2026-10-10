package notification

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationTypes(t *testing.T) {
	styles := Styles{
		Info:    lipgloss.NewStyle().Border(lipgloss.NormalBorder()),
		Success: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
		Warn:    lipgloss.NewStyle().Border(lipgloss.ThickBorder()),
		Error:   lipgloss.NewStyle().Border(lipgloss.DoubleBorder()),
	}
	for _, tt := range []struct {
		name     string
		cmd      tea.Cmd
		msgType  msgType
		corner   string
		lifetime time.Duration
	}{
		{"info", Info("notice"), msgTypeInfo, "┌", 2 * time.Second},
		{"success", Success("notice"), msgTypeSuccess, "╭", 2 * time.Second},
		{"warning", Warn("notice"), msgTypeWarn, "┏", 2 * time.Second},
		{"error", Error(errors.New("notice")), msgTypeError, "╔", 4 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				msg, ok := tt.cmd().(Msg)
				require.True(t, ok)
				assert.Equal(t, "notice", msg.Text)
				assert.Equal(t, tt.msgType, msg.msgType)

				m := New(WithStyles(styles))
				expiry, handled := m.Update(msg)
				require.True(t, handled)
				require.NotNil(t, expiry)
				base := strings.Repeat(".", 80) + "\n\n"
				rendered := m.Overlay(base)
				assert.Contains(t, rendered, "notice")
				assert.Contains(t, rendered, tt.corner)

				start := time.Now()
				_, handled = m.Update(expiry())
				assert.True(t, handled)
				assert.Equal(t, tt.lifetime, time.Since(start))
				assert.Equal(t, base, m.Overlay(base))
			})
		})
	}
}

func TestExpiryDoesNotDismissReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := New()
		first, handled := m.Update(Info("First")())
		require.True(t, handled)
		require.NotNil(t, first)
		second, handled := m.Update(Success("Second")())
		require.True(t, handled)
		require.NotNil(t, second)
		_, handled = m.Update(first())
		assert.True(t, handled, "expiry message was not handled")
		assert.Contains(t, m.Overlay(strings.Repeat(" ", 80)+"\n\n"), "Second", "old expiry dismissed the replacement")
		_, handled = m.Update(second())
		assert.True(t, handled)
		assert.Empty(t, m.message.Text, "current expiry did not clear the notification")
	})
}

func TestOverlayFitsAndSanitizes(t *testing.T) {
	for _, size := range []struct{ width, height int }{{80, 24}, {20, 8}, {11, 3}, {4, 3}, {80, 2}, {4, 2}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := New()
			cmd, handled := m.Update(Error(errors.New("\x1b[31mFailed\n" + strings.Repeat("界", 60)))())
			require.True(t, handled)
			require.NotNil(t, cmd, "error notification was not handled with an expiry command")
			base := strings.TrimSuffix(strings.Repeat(strings.Repeat(" ", size.width)+"\n", size.height), "\n")
			rendered := m.Overlay(base)
			assert.Equal(t, size.width, lipgloss.Width(rendered))
			assert.Equal(t, size.height, lipgloss.Height(rendered))
			assert.Equal(t, "Failed "+strings.Repeat("界", 60), m.message.Text)
			if size.width >= 8 && size.height >= 3 {
				text := sanitize.TerminalText(rendered)
				assert.Contains(t, text, "Fai", "notification text is missing")
				assert.Contains(t, text, "…", "long notification was not truncated")
			} else {
				assert.Equal(t, base, rendered, "overlay changed content on a terminal too small for a toast")
			}
		})
	}
}

func TestOverlayFitsCustomStyles(t *testing.T) {
	bordered := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	for _, tt := range []struct {
		name    string
		style   lipgloss.Style
		visible bool
	}{
		{"wide padding", bordered.Padding(0, 3), true},
		{"horizontal margins", bordered.Padding(0, 1).Margin(0, 3), true},
		{"vertical padding fits", bordered.Padding(2, 1), true},
		{"vertical margins fit", bordered.Padding(0, 1).Margin(2, 0), true},
		{"no room for text", bordered.Padding(0, 9), false},
		{"frame wider than content", bordered.Padding(0, 10), false},
		{"vertical padding too tall", bordered.Padding(3, 1), false},
		{"vertical margins too tall", bordered.Padding(0, 1).Margin(3, 0), false},
		{"explicit width too wide", bordered.Width(30), false},
		{"explicit height too tall", bordered.Height(12), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := New(WithStyles(Styles{Info: tt.style}))
			cmd, handled := m.Update(Info("notice " + strings.Repeat("界", 40))())
			require.True(t, handled)
			require.NotNil(t, cmd)
			base := strings.TrimSuffix(strings.Repeat(strings.Repeat(".", 20)+"\n", 8), "\n")
			rendered := m.Overlay(base)
			assert.Equal(t, 20, lipgloss.Width(rendered))
			assert.Equal(t, 8, lipgloss.Height(rendered))
			if !tt.visible {
				assert.Equal(t, base, rendered, "a notification that cannot fit should leave content unchanged")
				return
			}
			text := sanitize.TerminalText(rendered)
			assert.Contains(t, text, "notice")
			assert.Contains(t, text, "…")
			for _, corner := range []string{"┌", "┐", "└", "┘"} {
				assert.Contains(t, text, corner, "notification border should fit completely")
			}
		})
	}
}

func TestIgnoresUnrelatedMessages(t *testing.T) {
	m := New()
	cmd, handled := m.Update("unrelated")
	assert.Nil(t, cmd)
	assert.False(t, handled, "unrelated message was handled")
	assert.Nil(t, Error(nil), "nil error produced a command")
}
