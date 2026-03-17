package core

import "github.com/bpicode/tmus/internal/app/library"

// MetadataScope controls the amount of metadata to load.
type MetadataScope int

const (
	MetadataBasic MetadataScope = iota
	MetadataExtended
)

// TrackMetadataEvent reports tags for a track loaded in the background.
type TrackMetadataEvent struct {
	TrackID  uint64
	Path     string
	Scope    MetadataScope
	Metadata library.Metadata
	Err      error
}
