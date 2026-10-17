package footer

import (
	"testing"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestViewUsesSuppliedSnapshot(t *testing.T) {
	// Rendering status and volume needs no running app or state-reader stub.
	m := NewModel(Config{Theme: theme.Resolve(config.Default().TUI.Theme)})
	m.SetSize(80, 2)
	started := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	paused := core.State{
		Playlist: []core.Track{{Name: "Snapshot track"}},
		Playing:  0,
		Volume:   75,
		Playback: core.Playback{
			State: core.PlaybackPaused, StartedAt: started,
			PausedAt: started.Add(42 * time.Second), Duration: 3 * time.Minute,
		},
	}
	tests := []struct {
		name   string
		state  core.State
		status string
		volume string
	}{
		{name: "paused", state: paused, status: "Snapshot track [00:42/03:00]", volume: "75%"},
		{name: "stopped", state: core.State{Volume: 25}, status: "none", volume: "25%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := ansi.Strip(m.View(tt.state))
			assert.Contains(t, view, "Playing:  "+tt.status)
			assert.Contains(t, view, tt.volume)
		})
	}
}

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

func TestLayout(t *testing.T) {
	tests := []struct {
		name            string
		width           int
		availableHeight int
		statusHeight    int
		volumeHeight    int
		wantRows        int
		wantFooter      int
		wantStatus      bool
		wantVolume      bool
	}{
		{name: "two rows", width: 80, availableHeight: 6, statusHeight: 1, volumeHeight: 1, wantRows: 2, wantFooter: 4, wantStatus: true, wantVolume: true},
		{name: "one row", width: 80, availableHeight: 4, statusHeight: 1, volumeHeight: 1, wantRows: 1, wantFooter: 3, wantStatus: true, wantVolume: true},
		{name: "hidden by height", width: 80, availableHeight: 3, statusHeight: 1, volumeHeight: 1, wantRows: 0, wantFooter: 2, wantStatus: true, wantVolume: true},
		{name: "hidden by width", width: labelWidth, availableHeight: 6, statusHeight: 1, volumeHeight: 1, wantRows: 0, wantFooter: 2, wantStatus: true, wantVolume: true},
		{name: "spectrum-only footer", width: 80, availableHeight: 3, wantRows: 2, wantFooter: 2},
		{name: "status prioritized", width: 80, availableHeight: 1, statusHeight: 1, volumeHeight: 1, wantFooter: 1, wantStatus: true},
		{name: "no available height", width: 80, statusHeight: 1, volumeHeight: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{width: tt.width}
			layout := m.calculateLayout(tt.availableHeight, tt.statusHeight, tt.volumeHeight)
			assert.Equal(t, tt.wantRows, layout.spectrumRows)
			assert.Equal(t, tt.wantFooter, layout.height)
			assert.Equal(t, tt.wantStatus, layout.showStatus)
			assert.Equal(t, tt.wantVolume, layout.showVolume)
			assert.LessOrEqual(t, layout.height, tt.availableHeight)
		})
	}
}

func TestFormatLabel(t *testing.T) {
	assert.Equal(t, "Spectrum: ", formatLabel(spectrumLabel))
	assert.Equal(t, "Playing:  ", formatLabel(playingLabel))
	assert.Equal(t, "Volume:   ", formatLabel(volumeLabel))
}
