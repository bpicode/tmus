package library

import (
	"context"
	"io"
)

type entryType int

const (
	entryDir entryType = iota
	entryArchive
	entryMP3
	entryFLAC
	entryOGG
	entryOPUS
	entryOGA
	entryM4A
	entryMP4
	entryWAV
	entryStream
	entryOther
)

// FormatType identifies the decoder format for an AudioSource.
type FormatType int

const (
	// FormatMP3 identifies MPEG Layer III audio.
	FormatMP3 FormatType = iota
	// FormatFLAC identifies Free Lossless Audio Codec audio.
	FormatFLAC
	// FormatOGG identifies Ogg Vorbis audio.
	FormatOGG
	// FormatOPUS identifies Opus audio.
	FormatOPUS
	// FormatOGA identifies Ogg audio containers commonly using .oga.
	FormatOGA
	// FormatM4A identifies MPEG-4 audio commonly using .m4a.
	FormatM4A
	// FormatMP4 identifies MPEG-4 audio/video containers.
	FormatMP4
	// FormatWAV identifies Waveform Audio File Format audio.
	FormatWAV
	// FormatUnknown indicates that no decoder format could be inferred.
	FormatUnknown
)

// AudioSource is an opened playable byte stream and its decoder format.
type AudioSource struct {
	// Reader streams the encoded audio bytes. Callers own closing it.
	Reader io.ReadCloser
	// Format identifies the decoder to use for Reader.
	Format FormatType
}

// Entry represents a durable library item.
type Entry interface {
	// Path returns the durable path or URI for this entry.
	Path() string
	// Name returns the display name for this entry.
	Name() string
	// Hidden reports whether the entry should be treated as hidden.
	Hidden() bool
	// Open opens the entry as audio.
	Open(ctx context.Context) (AudioSource, error)
	// IsAudio reports whether Open can be attempted for this entry.
	IsAudio() bool
	// Parent returns the durable path or URI for the containing entry.
	Parent() string
	// IsDir reports whether this entry is a browsable directory.
	IsDir() bool
	// IsArchive reports whether this entry is an archive file.
	IsArchive() bool
	// FilesystemPath returns the local filesystem path when one exists.
	FilesystemPath() (string, bool)
	// CanBrowse reports whether BrowsePath can be passed to List.
	CanBrowse() bool
	// BrowsePath returns the path or URI to pass to List for this entry.
	BrowsePath() (string, bool)
}
