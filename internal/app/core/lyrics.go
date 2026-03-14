package core

import (
	"path/filepath"

	"github.com/bpicode/tmus/internal/app/lyrics"
)

// LyricsEvent reports lyrics loaded in the background.
type LyricsEvent struct {
	TrackID uint64
	Path    string
	Lyrics  lyrics.Lyrics
	Err     error
}

type lyricsRequest struct {
	TrackID uint64
	Path    string
}

func (a *App) requestLyrics(trackID uint64, path string) {
	if trackID == 0 || path == "" {
		return
	}
	req := lyricsRequest{
		TrackID: trackID,
		Path:    path,
	}
	select {
	case <-a.ctx.Done(): // Do nothing, drop request during shutdown.
	case a.lyricsQueue <- req: // Nothing else to do, just queue the request.
	}
}

func (a *App) lyricsWorker() {
	for {
		var req lyricsRequest
		select {
		case <-a.ctx.Done():
			return
		case req = <-a.lyricsQueue:
		}
		if a.ctx.Err() != nil {
			return
		}
		if req.Path == "" || req.TrackID == 0 {
			continue
		}
		if cached, ok := a.getCachedLyrics(req.Path); ok {
			if !a.emitLyricsEvent(LyricsEvent{
				TrackID: req.TrackID,
				Path:    req.Path,
				Lyrics:  cached,
			}) {
				return
			}
			continue
		}
		trackInfo := a.trackInfoForLyrics(req)
		result, err := a.lyricsResolver.Find(trackInfo)
		if a.ctx.Err() != nil {
			return
		}
		if err != nil {
			if !a.emitLyricsEvent(LyricsEvent{
				TrackID: req.TrackID,
				Path:    req.Path,
				Err:     err,
			}) {
				return
			}
			continue
		}
		a.putCachedLyrics(req.Path, result)
		if !a.emitLyricsEvent(LyricsEvent{
			TrackID: req.TrackID,
			Path:    req.Path,
			Lyrics:  result,
		}) {
			return
		}
	}
}

func (a *App) emitLyricsEvent(event LyricsEvent) bool {
	select {
	case <-a.ctx.Done():
		return false
	case a.lyricsChan <- event:
		return true
	}
}

func (a *App) trackInfoForLyrics(req lyricsRequest) lyrics.TrackInfo {
	info := lyrics.TrackInfo{
		Path: req.Path,
		Name: filepath.Base(req.Path),
	}
	if req.TrackID == 0 && req.Path == "" {
		return info
	}
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	for _, track := range a.state.Playlist {
		if req.TrackID != 0 {
			if track.ID != req.TrackID {
				continue
			}
		} else if req.Path != "" && track.Path != req.Path {
			continue
		}
		if track.Path != "" {
			info.Path = track.Path
		}
		if track.Name != "" {
			info.Name = track.Name
		}
		info.Artist = track.Artist
		info.Title = track.Title
		info.Album = track.Album
		info.Duration = track.Duration
		if info.Duration == 0 && a.state.PlayTrack == req.Path {
			info.Duration = a.state.PlayDuration
		}
		break
	}
	return info
}
