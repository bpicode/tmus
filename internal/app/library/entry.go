package library

import (
	"context"
	"io"
)

// EntryType identifies the kind or audio format of a library entry.
type EntryType int

const (
	EntryDir = iota
	EntryArchive
	EntryMP3
	EntryFLAC
	EntryOGG
	EntryOPUS
	EntryOGA
	EntryM4A
	EntryMP4
	EntryWAV
	EntryStream
	EntryOther
)

// FormatType identifies the audio decoder format for an opened source.
type FormatType int

const (
	FormatMP3 = iota
	FormatFLAC
	FormatOGG
	FormatOPUS
	FormatOGA
	FormatM4A
	FormatMP4
	FormatWAV
	FormatUnknown
)

// AudioSource is an opened audio byte stream and its decoder format.
type AudioSource struct {
	Reader io.ReadCloser
	Format FormatType
}

// Entry represents a browsable library item.
type Entry interface {
	Path() string
	Name() string
	Type() EntryType
	Hidden() bool
	Open(ctx context.Context) (AudioSource, error)
	IsAudio() bool
	Parent() string
	IsDir() bool
	IsArchive() bool
	CanBrowse() bool
	BrowsePath() (string, bool)
}
