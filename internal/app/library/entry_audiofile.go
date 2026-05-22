package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type audioFile struct {
	path string
	name string
}

func (a audioFile) Path() string {
	return a.path
}

func (a audioFile) Name() string {
	return a.name
}

func (a audioFile) entryType() entryType {
	return entryTypeFromPath(a.path)
}

func (a audioFile) Hidden() bool {
	return strings.HasPrefix(a.name, ".")
}

func (a audioFile) Open(_ context.Context) (AudioSource, error) {
	f, err := os.Open(a.path)
	if err != nil {
		return AudioSource{}, err
	}
	return AudioSource{Reader: f, Format: formatFromPath(a.path)}, nil
}

func (a audioFile) IsAudio() bool {
	return a.entryType() != entryOther
}

func (a audioFile) Parent() string {
	return filepath.Dir(a.path)
}

func (a audioFile) IsDir() bool {
	return false
}

func (a audioFile) IsArchive() bool {
	return false
}

func (a audioFile) FilesystemPath() (string, bool) {
	return a.path, true
}

func (a audioFile) CanBrowse() bool {
	return false
}

func (a audioFile) BrowsePath() (string, bool) {
	return "", false
}
