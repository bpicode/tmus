package playlist

import (
	"testing"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/player"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCollectsSpectrumCommand(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)

	_, cmd, stop := m.Update(core.StateEvent{Changes: core.StateChangePlaying})

	assert.False(t, stop)
	assert.NotNil(t, cmd)
}

func TestInitCollectsSpectrumCommandDuringPlayback(t *testing.T) {
	m, app := newSpectrumCommandTestModel(t)
	app.HandlePlayerEvent(player.Event{Type: player.EventTrackStarted, PlaybackID: 1})

	cmd := m.Init()

	assert.NotNil(t, cmd)
}

func newSpectrumCommandTestModel(t *testing.T) (*Model, *core.App) {
	t.Helper()
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	m := NewModel(Config{App: app, FPS: cfg.TUI.FPS})
	require.NotNil(t, m.footer)
	return m, app
}
