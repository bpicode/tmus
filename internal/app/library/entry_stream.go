package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type urlFile struct {
	path string
	name string
}

func (u urlFile) Path() string {
	return u.path
}

func (u urlFile) Name() string {
	return u.name
}

func (u urlFile) Type() EntryType {
	return EntryStream
}

func (u urlFile) Hidden() bool {
	return strings.HasPrefix(u.name, ".")
}

func (u urlFile) Open(ctx context.Context) (AudioSource, error) {
	file, err := os.Open(u.path)
	if err != nil {
		return AudioSource{}, err
	}
	defer file.Close()

	uri, err := ParseURLShortcut(file)
	if err != nil {
		return AudioSource{}, err
	}
	return openRemoteAudio(ctx, uri)
}

func (u urlFile) IsAudio() bool {
	return true
}

func (u urlFile) Parent() string {
	return filepath.Dir(u.path)
}

func (u urlFile) IsDir() bool {
	return false
}

func (u urlFile) IsArchive() bool {
	return false
}

func (u urlFile) CanBrowse() bool {
	return false
}

func (u urlFile) BrowsePath() (string, bool) {
	return "", false
}

type streamFile struct {
	path string
	name string
}

func (s streamFile) Path() string {
	return s.path
}

func (s streamFile) Name() string {
	return s.name
}

func (s streamFile) Type() EntryType {
	return EntryStream
}

func (s streamFile) Hidden() bool {
	return strings.HasPrefix(s.name, ".")
}

func (s streamFile) Open(ctx context.Context) (AudioSource, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return AudioSource{}, err
	}
	defer file.Close()

	uri, err := ParseStreamShortcut(file)
	if err != nil {
		return AudioSource{}, err
	}
	return openRemoteAudio(ctx, uri)
}

func (s streamFile) IsAudio() bool {
	return true
}

func (s streamFile) Parent() string {
	return filepath.Dir(s.path)
}

func (s streamFile) IsDir() bool {
	return false
}

func (s streamFile) IsArchive() bool {
	return false
}

func (s streamFile) CanBrowse() bool {
	return false
}

func (s streamFile) BrowsePath() (string, bool) {
	return "", false
}
