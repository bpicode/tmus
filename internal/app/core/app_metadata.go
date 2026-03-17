package core

import "github.com/bpicode/tmus/internal/app/library"

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

func readMetadataForScope(path string, scope MetadataScope) (Metadata, error) {
	switch scope {
	case MetadataExtended:
		return library.ReadMetadataExtended(path)
	default:
		return library.ReadMetadataBasic(path)
	}
}

func (a *App) readMetadataExtendedCached(path string) (Metadata, error) {
	if meta, ok := a.getCachedMetadata(path, MetadataExtended); ok {
		return meta, nil
	}
	meta, err := library.ReadMetadataExtended(path)
	if err != nil {
		return Metadata{}, err
	}
	a.putCachedMetadata(path, MetadataExtended, meta)
	return meta, nil
}

func (a *App) getCachedMetadata(path string, scope MetadataScope) (Metadata, bool) {
	if a == nil || a.metadataCache == nil {
		return Metadata{}, false
	}
	return a.metadataCache.get(path, scope)
}

func (a *App) putCachedMetadata(path string, scope MetadataScope, meta Metadata) {
	if a == nil || a.metadataCache == nil {
		return
	}
	a.metadataCache.put(path, scope, meta)
}

func (a *App) emitMetadataEvent(event TrackMetadataEvent) bool {
	select {
	case <-a.ctx.Done():
		return false
	case a.metadataChan <- event:
		return true
	}
}
