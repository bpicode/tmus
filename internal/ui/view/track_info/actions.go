package track_info

// MetadataRequest asks the parent to load extended metadata for a track.
type MetadataRequest struct {
	TrackID uint64
	Path    string
}
