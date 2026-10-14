package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveAtClearsUnusedTrack(t *testing.T) {
	tracks := []Track{
		{ID: 1, Path: "/music/one.flac", Title: "One"},
		{ID: 2, Path: "/music/two.flac", Title: "Two"},
		{ID: 3, Path: "/music/three.flac", Title: "Three"},
	}
	tests := []struct {
		name           string
		index          int
		want           []Track
		wantPlaying    int
		wantCursor     int
		removedPlaying bool
	}{
		{"first", 0, tracks[1:], 0, 1, false},
		{"playing", 1, []Track{tracks[0], tracks[2]}, -1, 1, true},
		{"last", 2, tracks[:2], 1, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playlist := make([]Track, len(tracks))
			copy(playlist, tracks)
			a := &App{
				state: State{Playlist: playlist, Playing: 1, Cursor: 2},
				queue: LinearStrategy{},
			}

			removedPlaying, removed := a.removeAt(tt.index)

			assert.True(t, removed)
			assert.Equal(t, tt.removedPlaying, removedPlaying)
			assert.Equal(t, tt.want, a.state.Playlist)
			assert.Equal(t, tt.wantPlaying, a.state.Playing)
			assert.Equal(t, tt.wantCursor, a.state.Cursor)
			// The original slice still exposes the vacated backing-array slot.
			assert.Zero(t, playlist[len(playlist)-1], "unused slot retains track data")
		})
	}
}
