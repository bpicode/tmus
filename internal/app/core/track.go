package core

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
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
	return strings.HasPrefix(t.Path, "http://") || strings.HasPrefix(t.Path, "https://")
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
		if entry, err := library.EntryFromPath(t.Path); err == nil {
			return entry.Name()
		}
		return fallbackTrackName(t.Path)
	}
	return ""
}

func fallbackTrackName(value string) string {
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Path != "" {
		name := path.Base(parsed.Path)
		if name != "/" && name != "." {
			return name
		}
		return value
	}
	return filepath.Base(value)
}
