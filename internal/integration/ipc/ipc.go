package ipc

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
)

var (
	// ErrNoServer indicates no running tmus instance was found.
	ErrNoServer = errors.New("no running tmus instance")
	// ErrAlreadyRunning indicates a running tmus instance already owns the IPC endpoint.
	ErrAlreadyRunning = errors.New("tmus instance already running")
	// ErrNotSupported indicates IPC is not supported on this platform.
	ErrNotSupported = errors.New("ipc not supported")
)

type request struct {
	Paths []string `json:"paths"`
}

type response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func buildTracks(paths []string) []core.Track {
	if len(paths) == 0 {
		return nil
	}
	tracks := make([]core.Track, 0, len(paths))
	for _, value := range paths {
		entry, err := library.EntryFromPath(normalizeInputPath(value))
		if err == nil && entry.IsAudio() {
			tracks = append(tracks, core.Track{
				Name: entry.Name(),
				Path: entry.Path(),
			})
		}
	}
	return tracks
}

func normalizeInputPath(value string) string {
	if value == "" || strings.Contains(value, "://") {
		return value
	}
	path := filepath.Clean(value)
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}
