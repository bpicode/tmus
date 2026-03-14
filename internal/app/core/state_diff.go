package core

// DiffState compares two state snapshots and returns the changes.
func DiffState(prev, next State) StateChange {
	changes := StateChangeNone

	if prev.QueueMode != next.QueueMode {
		changes |= StateChangeQueue
	}
	if prev.Volume != next.Volume {
		changes |= StateChangeVolume
	}
	if prev.Cursor != next.Cursor {
		changes |= StateChangeSelection
	}
	if prev.Playing != next.Playing || prev.PlayTrack != next.PlayTrack {
		changes |= StateChangePlaying | StateChangeMetadata
	}
	if prev.PlayState != next.PlayState ||
		prev.PlayStart != next.PlayStart ||
		prev.PlayDuration != next.PlayDuration ||
		prev.PausedAt != next.PausedAt ||
		prev.PausedFor != next.PausedFor {
		changes |= StateChangePlayback
	}
	if prev.PlaylistErr != next.PlaylistErr {
		changes |= StateChangeError
	}

	if len(prev.Playlist) != len(next.Playlist) {
		changes |= StateChangePlaylist
		return changes
	}

	for i := range prev.Playlist {
		before := prev.Playlist[i]
		after := next.Playlist[i]
		if before.ID != after.ID || before.Path != after.Path || before.Name != after.Name {
			changes |= StateChangePlaylist
		}
		if before.Artist != after.Artist || before.Title != after.Title || before.Album != after.Album {
			changes |= StateChangeMetadata
		}
		if changes&(StateChangePlaylist|StateChangeMetadata) == StateChangePlaylist|StateChangeMetadata {
			break
		}
	}

	return changes
}
