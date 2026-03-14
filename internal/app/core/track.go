package core

import (
	"path/filepath"
	"time"
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
		return filepath.Base(t.Path)
	}
	return ""
}
