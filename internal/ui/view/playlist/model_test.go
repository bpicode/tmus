package playlist

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/player"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
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

func TestWindowSizeUpdatesChildLayout(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)

	const width = 80
	const height = 10
	_, _, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})

	innerHeight := height - m.styles.panelUnfocused.GetVerticalFrameSize()
	headerHeight := len(m.headerLines(m.app.State(), m.layout.innerWidth))
	assert.Equal(t, width-m.styles.panelUnfocused.GetHorizontalFrameSize(), m.layout.innerWidth)
	assert.Equal(t, innerHeight, headerHeight+m.layout.bodyHeight+m.layout.footerHeight)
	assert.Equal(t, m.layout.innerWidth, m.list.Width())
	assert.Equal(t, m.layout.bodyHeight, m.list.Height())
	assert.Equal(t, m.footer.Height()+1, m.layout.footerHeight)
}

func TestViewDoesNotResizeChildren(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)
	_, _, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	wantListWidth := m.list.Width()
	wantListHeight := m.list.Height()
	wantFooterHeight := m.footer.Height()

	_ = m.View()

	assert.Equal(t, wantListWidth, m.list.Width())
	assert.Equal(t, wantListHeight, m.list.Height())
	assert.Equal(t, wantFooterHeight, m.footer.Height())
}

func TestHeaderLinesIncludePlaylistError(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)

	assert.Len(t, m.headerLines(core.State{}, 80), 3)
	assert.Len(t, m.headerLines(core.State{PlaylistErr: errors.New("test")}, 80), 4)
}

func TestBodyLinesFillEmptyBody(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)
	m.layout.bodyHeight = 3

	lines := m.bodyLines()

	assert.Len(t, lines, 3)
	assert.Contains(t, lines[0], "(empty)")
}

func newSpectrumCommandTestModel(t *testing.T) (*Model, *core.App) {
	t.Helper()
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	m := NewModel(Config{Theme: theme.Resolve(cfg.TUI.Theme), App: app, FPS: cfg.TUI.FPS})
	require.NotNil(t, m.footer)
	return m, app
}
