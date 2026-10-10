package view

import (
	"errors"
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
				m := &Model{notifications: notification.New(notification.DefaultStyles())}
				_, expiry := m.Update(tt.wrap(notification.Error(errors.New("Could not add file")))())
				require.NotNil(t, expiry)
				assert.Contains(t, m.notifications.Overlay("                  "+"\n"+"                  "+"\n"+"                  "), "Could not")
				_, cmd := m.Update(expiry())
				assert.Nil(t, cmd)
				base := "                  "
				assert.Equal(t, base, m.notifications.Overlay(base))
			})
		})
	}
}
