package view

import (
	"errors"
	"strings"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/ui/components/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChildNotificationsReachRoot(t *testing.T) {
	for _, tt := range []struct {
		name string
		wrap func(tea.Cmd) tea.Cmd
	}{
		{"browser", browserCmd},
		{"playlist", playlistCmd},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				m := &Model{notifications: notification.New()}
				base := strings.TrimSuffix(strings.Repeat(strings.Repeat(".", 40)+"\n", 4), "\n")
				_, expiry := m.Update(tt.wrap(notification.Error(errors.New("could not add file")))())
				require.NotNil(t, expiry)
				assert.Contains(t, m.notifications.Overlay(base), "could not add file")
				_, cmd := m.Update(expiry())
				assert.Nil(t, cmd)
				assert.Equal(t, base, m.notifications.Overlay(base))
			})
		})
	}
}
