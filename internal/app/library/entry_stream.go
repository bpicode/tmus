package library

import (
	"context"
	"path/filepath"
	"strings"
)

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
	return EntryURL
}

func (s streamFile) Hidden() bool {
	return strings.HasPrefix(s.name, ".")
}

func (s streamFile) Open(ctx context.Context) (AudioSource, error) {
	uri, err := ParseStreamFile(s.path)
	if err != nil {
		return AudioSource{}, err
	}
	source, err := NewHTTPResolver().Resolve(ctx, uri)
	if err != nil {
		return AudioSource{}, err
	}
	return AudioSource{Reader: source.Reader, Format: formatFromExt(source.Ext)}, nil
}

func (s streamFile) IsAudio() bool {
	return true
}

func (s streamFile) Parent() string {
	return filepath.Dir(s.path)
}
