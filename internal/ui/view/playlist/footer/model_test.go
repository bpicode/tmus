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
