package core

import (
	"fmt"
	"time"
)

// Playback describes the active playback source and its current presentation state.
type Playback struct {
	State      PlaybackState
	Source     string
	StartedAt  time.Time
	Duration   time.Duration
	PausedAt   time.Time
	PausedFor  time.Duration
	NowPlaying NowPlaying
}

// Elapsed returns the elapsed time within the active playback source.
func (p Playback) Elapsed() time.Duration {
	if p.State == PlaybackStopped || p.StartedAt.IsZero() {
		return 0
	}
	elapsed := time.Since(p.StartedAt) - p.PausedFor
	if p.State == PlaybackPaused && !p.PausedAt.IsZero() {
		elapsed = p.PausedAt.Sub(p.StartedAt) - p.PausedFor
	}
	return elapsed
}

// NowPlaying describes the logical media item currently presented by a playback source.
type NowPlaying struct {
	DisplayTitle string
	Artist       string
	Title        string
	Album        string
	SourceName   string
}

// DisplayName returns the best available label for the logical media item.
func (n NowPlaying) DisplayName() string {
	if n.Artist != "" && n.Title != "" {
		return fmt.Sprintf("%s - %s", n.Artist, n.Title)
	}
	if n.DisplayTitle != "" {
		return n.DisplayTitle
	}
	if n.Title != "" {
		return n.Title
	}
	return n.SourceName
}
