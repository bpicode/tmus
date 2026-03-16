package core

import (
	"path/filepath"

	"github.com/bpicode/tmus/internal/app/library"
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

func (a *App) enqueueMetadataRead(track Track) {
	if track.Path == "" {
		return
	}
	if track.Artist != "" || track.Title != "" || track.Album != "" {
		return
	}
	a.requestMetadata(track.ID, track.Path, MetadataBasic)
}

func (a *App) requestMetadata(trackID uint64, path string, scope MetadataScope) {
	if trackID == 0 || path == "" {
		return
	}
	req := metadataRequest{
		TrackID: trackID,
		Path:    path,
		Scope:   scope,
	}
	select {
	case <-a.ctx.Done(): // Do nothing, drop request during shutdown.
	case a.metadataQueue <- req: // Nothing else to do, just queue the request.
	}
}

// HandleMetadataEvent updates playlist metadata when tags are read in the background.
func (a *App) HandleMetadataEvent(event TrackMetadataEvent) {
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

func (a *App) metadataWorker() {
	for {
		var req metadataRequest
		select {
		case <-a.ctx.Done():
			return // Stop worker on shutdown.
		case req = <-a.metadataQueue: // Receive item from queue to work on.
		}
		if a.ctx.Err() != nil {
			return
		}
		if req.Path == "" || req.TrackID == 0 {
			continue
		}
		if meta, ok := a.getCachedMetadata(req.Path, req.Scope); ok {
			if req.Scope == MetadataBasic && meta.Artist == "" && meta.Title == "" && meta.Album == "" {
				continue
			}
			if !a.emitMetadataEvent(TrackMetadataEvent{
				TrackID:  req.TrackID,
				Path:     req.Path,
				Scope:    req.Scope,
				Metadata: meta,
			}) {
				return
			}
			continue
		}
		meta, err := readMetadataForScope(req.Path, req.Scope)
		if err != nil {
			// Only surface errors for explicit extended requests to avoid spamming
			// subscribers with "no metadata" during background basic reads.
			if req.Scope == MetadataExtended {
				if !a.emitMetadataEvent(TrackMetadataEvent{
					TrackID: req.TrackID,
					Path:    req.Path,
					Scope:   req.Scope,
					Err:     err,
				}) {
					return
				}
			}
			continue
		}
		if req.Scope == MetadataBasic && meta.Artist == "" && meta.Title == "" && meta.Album == "" {
			continue
		}
		a.putCachedMetadata(req.Path, req.Scope, meta)
		if !a.emitMetadataEvent(TrackMetadataEvent{
			TrackID:  req.TrackID,
			Path:     req.Path,
			Scope:    req.Scope,
			Metadata: meta,
		}) {
			return
		}
	}
}

func (a *App) emitMetadataEvent(event TrackMetadataEvent) bool {
	select {
	case <-a.ctx.Done():
		return false
	case a.metadataChan <- event:
		return true
	}
}

func readMetadataForScope(path string, scope MetadataScope) (Metadata, error) {
	switch scope {
	case MetadataExtended:
		return library.ReadMetadataExtended(path)
	default:
		return library.ReadMetadataBasic(path)
	}
}
