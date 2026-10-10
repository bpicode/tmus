package notification

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
)

func TestExpiryDoesNotDismissReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := New(DefaultStyles())
		first, _ := m.Update(Info("First")())
		second, _ := m.Update(Success("Second")())
		if _, handled := m.Update(first()); !handled {
			t.Fatal("expiry message was not handled")
		}
		if !strings.Contains(m.Overlay(strings.Repeat(" ", 80)+"\n\n"), "Second") {
			t.Fatal("old expiry dismissed the replacement")
		}
		if _, handled := m.Update(second()); !handled || m.message.Text != "" {
			t.Fatal("current expiry did not clear the notification")
		}
	})
}

func TestOverlayFitsAndSanitizes(t *testing.T) {
	for _, size := range []struct{ width, height int }{{80, 24}, {20, 8}, {8, 3}, {4, 2}} {
		t.Run(fmt.Sprint(size.width), func(t *testing.T) {
			m := New(DefaultStyles())
			cmd, handled := m.Update(Error(errors.New("\x1b[31mFailed\n" + strings.Repeat("界", 60)))())
			if !handled || cmd == nil {
				t.Fatal("error notification was not handled with an expiry command")
			}
			base := strings.TrimSuffix(strings.Repeat(strings.Repeat(" ", size.width)+"\n", size.height), "\n")
			rendered := m.Overlay(base)
			if w, h := lipgloss.Width(rendered), lipgloss.Height(rendered); w != size.width || h != size.height {
				t.Fatalf("overlay dimensions = %dx%d, want %dx%d", w, h, size.width, size.height)
			}
			if strings.ContainsAny(m.message.Text, "\x1b\n") {
				t.Fatalf("notification was not sanitized: %q", m.message.Text)
			}
			if size.width >= 8 {
				if !strings.Contains(sanitize.TerminalText(rendered), "!") {
					t.Fatal("error indicator is missing")
				}
			} else if rendered != base {
				t.Fatal("overlay changed content on a terminal too small for a toast")
			}
		})
	}
}

func TestIgnoresUnrelatedMessages(t *testing.T) {
	m := New(Styles{})
	if cmd, handled := m.Update("unrelated"); cmd != nil || handled {
		t.Fatal("unrelated message was handled")
	}
	if Error(nil) != nil {
		t.Fatal("nil error produced a command")
	}
}
