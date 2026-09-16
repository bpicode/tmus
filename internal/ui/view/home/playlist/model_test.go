package playlist

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/player"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpectrumRowCount(t *testing.T) {
	tests := []struct {
		capacity int
		want     int
	}{
		{capacity: 0, want: 0},
		{capacity: 1, want: 0},
		{capacity: 2, want: 1},
		{capacity: 3, want: 2},
		{capacity: 20, want: 2},
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, spectrumRowCount(tt.capacity), "capacity %d", tt.capacity)
	}
}

func TestSpectrumLayout(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		header     int
		status     string
		volume     string
		wantRows   int
		wantFooter int
	}{
		{name: "two rows", width: 80, height: 10, header: 3, status: "status", volume: "volume", wantRows: 2, wantFooter: 5},
		{name: "one row", width: 80, height: 8, header: 3, status: "status", volume: "volume", wantRows: 1, wantFooter: 4},
		{name: "hidden by height", width: 80, height: 7, header: 3, status: "status", volume: "volume", wantRows: 0, wantFooter: 3},
		{name: "hidden by width", width: footerLabelWidth, height: 10, header: 3, status: "status", volume: "volume", wantRows: 0, wantFooter: 3},
		{name: "spectrum-only footer", width: 80, height: 7, header: 3, wantRows: 2, wantFooter: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, footer := spectrumLayout(tt.width, tt.height, tt.header, tt.status, tt.volume)
			assert.Equal(t, tt.wantRows, rows)
			assert.Equal(t, tt.wantFooter, footer)
			assert.GreaterOrEqual(t, tt.height-tt.header-footer, 1)
		})
	}
}

func TestFormatFooterLabel(t *testing.T) {
	assert.Equal(t, "Spectrum: ", formatFooterLabel(spectrumLabel))
	assert.Equal(t, "Playing:  ", formatFooterLabel(playingLabel))
	assert.Equal(t, "Volume:   ", formatFooterLabel(volumeLabel))
}

func TestSpectrumViewAlignsLabelWithBottomRow(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)

	lines := m.spectrumView(20, 2)

	require.Len(t, lines, 2)
	assert.Equal(t, strings.Repeat(" ", 20), lines[0])
	assert.Equal(t, "Spectrum: "+strings.Repeat(" ", 10), lines[1])
	assert.Equal(t, 20, lipgloss.Width(lines[0]))
	assert.Equal(t, 20, lipgloss.Width(lines[1]))
}

func TestSpectrumViewLabelsSingleRow(t *testing.T) {
	m, _ := newSpectrumCommandTestModel(t)

	lines := m.spectrumView(20, 1)

	require.Len(t, lines, 1)
	assert.Equal(t, "Spectrum: "+strings.Repeat(" ", 10), lines[0])
}

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
	require.NotNil(t, m.spectrum)
	return m, app
}
