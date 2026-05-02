package core

import (
	"time"

	"github.com/bpicode/tmus/internal/app/library"
)

// Track represents a playable file.
type Track struct {
	ID       uint64
	Name     string
	Path     string
	Artist   string
	Title    string
	Album    string
	Duration time.Duration
}

// IsRemote reports whether the track is hosted externally (e.g., HTTP/HTTPS).
func (t Track) IsRemote() bool {
	return library.IsRemote(t.Path)
}

// DisplayName returns the most user-friendly track name available.
func (t Track) DisplayName() string {
	if t.Artist != "" && t.Title != "" {
		return t.Artist + " - " + t.Title
	}
	if t.Title != "" {
		return t.Title
	}
	if t.Name != "" {
		return t.Name
	}
	if t.Path != "" {
		return library.BaseName(t.Path)
	}
	return ""
}
