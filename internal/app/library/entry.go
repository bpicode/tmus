package library

import (
	"context"
	"io"
)

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

type AudioSource struct {
	Reader io.ReadCloser
	Format FormatType
}

type Entry2 interface {
	Path() string
	Name() string
	Type() EntryType
	Hidden() bool
	Open(ctx context.Context) (AudioSource, error)
	IsAudio() bool
	Parent() string
}
