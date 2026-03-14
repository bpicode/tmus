package core

import "github.com/bpicode/tmus/internal/app/library"

// MetadataScope controls the amount of metadata to load.
type MetadataScope int

const (
	MetadataBasic MetadataScope = iota
	MetadataExtended
)

// Metadata represents supported audio tag fields.
type Metadata = library.Metadata

// Picture represents embedded artwork data.
type Picture = library.Picture

// TrackMetadataEvent reports tags for a track loaded in the background.
type TrackMetadataEvent struct {
	TrackID  uint64
	Path     string
	Scope    MetadataScope
	Metadata Metadata
	Err      error
}
