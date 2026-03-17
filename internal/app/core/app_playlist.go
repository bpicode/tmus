package core

import (
	"path/filepath"
)

func (a *App) addTrack(track Track) {
	if track.Path == "" {
		return
	}
	if track.Name == "" {
		track.Name = filepath.Base(track.Path)
	}
	if track.ID == 0 {
		track.ID = a.nextID()
	}
	a.state.Playlist = append(a.state.Playlist, track)
	if a.state.Cursor == -1 {
		a.state.Cursor = len(a.state.Playlist) - 1
	}
	a.state.PlaylistErr = nil
	a.enqueueMetadataRead(track)
}

func (a *App) clearPlaylist() {
	a.state.Playlist = nil
	a.state.Playing = -1
	a.state.Cursor = -1
	a.state.PlaylistErr = nil
	a.queue.Reset()
}

func (a *App) stop() {
	a.state.Playing = -1
	a.state.PlaylistErr = nil
}

func (a *App) removeAt(index int) (removedPlaying bool, removed bool) {
	if index < 0 || index >= len(a.state.Playlist) {
		return false, false
	}
	removedPlaying = index == a.state.Playing
	a.state.Playlist = append(a.state.Playlist[:index], a.state.Playlist[index+1:]...)
	a.queue.Reset()
	if len(a.state.Playlist) == 0 {
		a.state.Playing = -1
		a.state.Cursor = -1
		return removedPlaying, true
	}
	if index < a.state.Playing {
		a.state.Playing--
	} else if removedPlaying {
		a.state.Playing = -1
	}
	if index < a.state.Cursor {
		a.state.Cursor--
	} else if index == a.state.Cursor {
		a.state.Cursor = clamp(a.state.Cursor, 0, len(a.state.Playlist)-1)
	}
	return removedPlaying, true
}

func (a *App) moveUp() {
	i := a.state.Cursor
	if i <= 0 || i >= len(a.state.Playlist) {
		return
	}
	a.state.Playlist[i], a.state.Playlist[i-1] = a.state.Playlist[i-1], a.state.Playlist[i]
	switch a.state.Playing {
	case i:
		a.state.Playing = i - 1
	case i - 1:
		a.state.Playing = i
	}
	a.state.Cursor = i - 1
}

func (a *App) moveDown() {
	i := a.state.Cursor
	if i < 0 || i >= len(a.state.Playlist)-1 {
		return
	}
	a.state.Playlist[i], a.state.Playlist[i+1] = a.state.Playlist[i+1], a.state.Playlist[i]
	switch a.state.Playing {
	case i:
		a.state.Playing = i + 1
	case i + 1:
		a.state.Playing = i
	}
	a.state.Cursor = i + 1
}

func (a *App) selectUp() {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	if a.state.Cursor == -1 {
		a.state.Cursor = 0
		return
	}
	a.state.Cursor = clamp(a.state.Cursor-1, 0, len(a.state.Playlist)-1)
}

func (a *App) selectDown() {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	if a.state.Cursor == -1 {
		a.state.Cursor = 0
		return
	}
	a.state.Cursor = clamp(a.state.Cursor+1, 0, len(a.state.Playlist)-1)
}

func (a *App) selectTop() {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	a.state.Cursor = 0
}

func (a *App) selectBottom() {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	a.state.Cursor = len(a.state.Playlist) - 1
}

func (a *App) selectPageUp(count int) {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	if a.state.Cursor == -1 {
		a.state.Cursor = 0
		return
	}
	if count < 1 {
		count = 1
	}
	a.state.Cursor = clamp(a.state.Cursor-count, 0, len(a.state.Playlist)-1)
}

func (a *App) selectPageDown(count int) {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	if a.state.Cursor == -1 {
		a.state.Cursor = 0
		return
	}
	if count < 1 {
		count = 1
	}
	a.state.Cursor = clamp(a.state.Cursor+count, 0, len(a.state.Playlist)-1)
}

func (a *App) selectIndex(index int) {
	if len(a.state.Playlist) == 0 {
		a.state.Cursor = -1
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(a.state.Playlist) {
		index = len(a.state.Playlist) - 1
	}
	a.state.Cursor = index
}

func (a *App) playFromCursor() int {
	if a.state.Cursor < 0 || a.state.Cursor >= len(a.state.Playlist) {
		return -1
	}
	a.state.Playing = a.state.Cursor
	a.state.PlaylistErr = nil
	return a.state.Playing
}

func (a *App) next() int {
	qi := QueueInput{PlaylistLen: len(a.state.Playlist), Playing: a.state.Playing}
	return a.applyQueueDecision(a.queue.Next(qi))
}

func (a *App) prev() int {
	qi := QueueInput{PlaylistLen: len(a.state.Playlist), Playing: a.state.Playing}
	return a.applyQueueDecision(a.queue.Prev(qi))
}

func (a *App) applyQueueDecision(d QueueDecision) int {
	if d.Index >= 0 {
		a.state.Playing = d.Index
		a.state.Cursor = d.Index
		a.state.PlaylistErr = nil
		return d.Index
	}
	if d.Stop {
		a.state.Playing = -1
	}
	return -1
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (a *App) nextID() uint64 {
	a.nextTrackID++
	return a.nextTrackID
}

func (a *App) updatePlayList(event TrackMetadataEvent) {
	if event.TrackID == 0 || event.Err != nil {
		return
	}
	a.stateMu.Lock()
	updated := false
	for i := range a.state.Playlist {
		track := &a.state.Playlist[i]
		if track.ID != event.TrackID {
			continue
		}
		if event.Path != "" && track.Path != event.Path {
			break
		}
		if track.Artist != "" || track.Title != "" || track.Album != "" || track.Duration > 0 {
			break
		}
		if event.Metadata.Artist == "" && event.Metadata.Title == "" && event.Metadata.Album == "" && event.Metadata.Duration == 0 {
			break
		}
		track.Artist = event.Metadata.Artist
		track.Title = event.Metadata.Title
		track.Album = event.Metadata.Album
		track.Duration = event.Metadata.Duration
		updated = true
		break
	}
	a.stateMu.Unlock()
	if updated {
		a.broadcastStateEvent(StateEvent{
			Source:  StateEventMetadata,
			Changes: StateChangeMetadata,
		})
	}
}
