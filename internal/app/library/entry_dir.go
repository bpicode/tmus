package library

import (
	"context"
	"path/filepath"
	"strings"
)

type dirEntry struct {
	path string
	name string
}

func (d dirEntry) Path() string {
	return d.path
}

func (d dirEntry) Name() string {
	return d.name
}

func (d dirEntry) Type() EntryType {
	return EntryDir
}

func (d dirEntry) Hidden() bool {
	return strings.HasPrefix(d.name, ".")
}

func (d dirEntry) Open(_ context.Context) (AudioSource, error) {
	return AudioSource{}, errNotAudio
}

func (d dirEntry) IsAudio() bool {
	return false
}

func (d dirEntry) Parent() string {
	return filepath.Dir(d.path)
}

func (d dirEntry) IsDir() bool {
	return true
}

func (d dirEntry) IsArchive() bool {
	return false
}

func (d dirEntry) CanBrowse() bool {
	return true
}

func (d dirEntry) BrowsePath() (string, bool) {
	return d.path, true
}
