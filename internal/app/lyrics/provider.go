package lyrics

import "time"

// Provider resolves lyrics for a track.
type Provider interface {
	name() Source
	find(track TrackInfo) (Lyrics, error)
}

// Source identifies where lyrics were found.
type Source string

// TrackInfo provides minimal context for resolving lyrics.
type TrackInfo struct {
	Path     string
	Name     string
	Artist   string
	Title    string
	Album    string
	Duration time.Duration
}

// Lyrics holds parsed lyrics data.
type Lyrics struct {
	Lines      []Line
	Timed      bool
	Raw        string
	Source     Source
	SourcePath string
}

// Line represents a single lyric line.
type Line struct {
	Text    string
	Time    time.Duration
	HasTime bool
}
