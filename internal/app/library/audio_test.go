package library

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAudio(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "MP3 file", path: "song.mp3", want: true},
		{name: "WAV file", path: "sound.wav", want: true},
		{name: "FLAC file", path: "album.flac", want: true},
		{name: "OPUS file", path: "voice.opus", want: true},
		{name: "OGG file", path: "music.ogg", want: true},
		{name: "OGA file", path: "audio.oga", want: true},
		{name: "M4A file", path: "track.m4a", want: true},
		{name: "MP4 file", path: "video.mp4", want: true},
		{name: "MP3 file (full path)", path: "/home/user/my-lib/song.mp3", want: true},
		{name: "MP3 file (archive path)", path: "arch://zip:/home/user/my-lib/album.zip::song.mp3", want: true},
		{name: "Unsupported extension", path: "document.txt", want: false},
		{name: "No extension", path: "file", want: false},
		{name: "Archive", path: "a.rar", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAudio(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
