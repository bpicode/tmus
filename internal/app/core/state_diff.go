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
	if prev.Playing != next.Playing || prev.Playback.Source != next.Playback.Source {
		changes |= StateChangePlaying | StateChangeMetadata
	}
	if prev.Playback.State != next.Playback.State ||
		prev.Playback.StartedAt != next.Playback.StartedAt ||
		prev.Playback.Duration != next.Playback.Duration ||
		prev.Playback.PausedAt != next.Playback.PausedAt ||
		prev.Playback.PausedFor != next.Playback.PausedFor {
		changes |= StateChangePlayback
	}
	if prev.Playback.NowPlaying != next.Playback.NowPlaying {
		changes |= StateChangeMetadata
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
