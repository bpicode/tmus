package footer

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestLayout(t *testing.T) {
	tests := []struct {
		name            string
		width           int
		availableHeight int
		status          string
		volume          string
		wantRows        int
		wantFooter      int
	}{
		{name: "two rows", width: 80, availableHeight: 7, status: "status", volume: "volume", wantRows: 2, wantFooter: 5},
		{name: "one row", width: 80, availableHeight: 5, status: "status", volume: "volume", wantRows: 1, wantFooter: 4},
		{name: "hidden by height", width: 80, availableHeight: 4, status: "status", volume: "volume", wantRows: 0, wantFooter: 3},
		{name: "hidden by width", width: labelWidth, availableHeight: 7, status: "status", volume: "volume", wantRows: 0, wantFooter: 3},
		{name: "spectrum-only footer", width: 80, availableHeight: 4, wantRows: 2, wantFooter: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{width: tt.width}
			rows, footer := m.layout(tt.availableHeight, tt.status, tt.volume)
			assert.Equal(t, tt.wantRows, rows)
			assert.Equal(t, tt.wantFooter, footer)
			assert.GreaterOrEqual(t, tt.availableHeight-footer, 1)
		})
	}
}

func TestFormatLabel(t *testing.T) {
	assert.Equal(t, "Spectrum: ", formatLabel(spectrumLabel))
	assert.Equal(t, "Playing:  ", formatLabel(playingLabel))
	assert.Equal(t, "Volume:   ", formatLabel(volumeLabel))
}
