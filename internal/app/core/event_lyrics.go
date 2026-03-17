package core

import "github.com/bpicode/tmus/internal/app/lyrics"

// LyricsEvent reports lyrics loaded in the background.
type LyricsEvent struct {
	TrackID uint64
	Path    string
	Lyrics  lyrics.Lyrics
	Err     error
}
