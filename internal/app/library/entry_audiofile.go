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

func (a audioFile) Type() EntryType {
	ext := filepath.Ext(a.path)
	ext = strings.TrimSpace(ext)
	ext = strings.ToLower(ext)
	switch ext {
	case ".mp3":
		return EntryMP3
	case ".flac":
		return EntryFLAC
	case ".ogg":
		return EntryOGG
	case ".opus":
		return EntryOPUS
	case ".oga":
		return EntryOGA
	case ".m4a":
		return EntryM4A
	case ".mp4":
		return EntryMP4
	case ".wav":
		return EntryWAV
	default:
		return EntryOther
	}
}

func (a audioFile) Hidden() bool {
	return strings.HasPrefix(a.name, ".")
}

func (a audioFile) Open(_ context.Context) (AudioSource, error) {
	f, err := os.Open(a.path)
	if err != nil {
		return AudioSource{}, err
	}
	var format FormatType
	switch strings.ToLower(filepath.Ext(a.path)) {
	case ".mp3":
		format = FormatMP3
	case ".flac":
		format = FormatFLAC
	case ".ogg":
		format = FormatOGG
	case ".opus":
		format = FormatOPUS
	case ".oga":
		format = FormatOGA
	case ".m4a":
		format = FormatM4A
	case ".mp4":
		format = FormatMP4
	case ".wav":
		format = FormatWAV
	default:
		format = FormatUnknown
	}
	return AudioSource{Reader: f, Format: format}, nil
}

func (a audioFile) IsAudio() bool {
	return a.Type() != EntryOther
}

func (a audioFile) Parent() string {
	return filepath.Dir(a.path)
}
