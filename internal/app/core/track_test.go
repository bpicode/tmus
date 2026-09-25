package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackIsRemote(t *testing.T) {
	tests := []struct {
		track Track
		want  bool
	}{
		{
			track: Track{Path: "http://stream.me/radio.pls"},
			want:  true,
		},
		{
			track: Track{Path: "https://stream.me/radio.pls"},
			want:  true,
		},
		{
			track: Track{Path: "/path/to/song.mp3"},
			want:  false,
		},
		{
			track: Track{Path: "arch://zip:/path/to/archive.zip::song.mp3"},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s -> %t", tt.track.Path, tt.want), func(t *testing.T) {
			got := tt.track.IsRemote()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTrackDiplayName(t *testing.T) {
	tests := []struct {
		name  string
		track Track
		want  string
	}{
		{
			name:  "Artist and title",
			track: Track{Artist: "Britney Shears", Title: "Maybe One More Line"},
			want:  "Britney Shears - Maybe One More Line",
		},
		{
			name:  "Just title",
			track: Track{Title: "Maybe One More Line"},
			want:  "Maybe One More Line",
		},
		{
			name:  "Just name",
			track: Track{Name: "song.mp3"},
			want:  "song.mp3",
		},
		{
			name:  "Just archive path",
			track: Track{Path: "arch://zip:/path/to/archive.zip::song.mp3"},
			want:  "song.mp3",
		},
		{
			name:  "Just filesystem path",
			track: Track{Path: "/path/to/song.mp3"},
			want:  "song.mp3",
		},
		{
			name:  "Just stream URL",
			track: Track{Path: "https://stream.me/radio.pls"},
			want:  "radio.pls",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.track.DisplayName()
			assert.Equal(t, tt.want, got)
		})
	}
}
