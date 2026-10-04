package view

import (
	"os"
	"path/filepath"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
)

func initialBrowserDir(lib *library.Library, startDir, savedDir string) string {
	cwd := startDir
	if cwd == "" {
		cwd = savedDir
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if entry, err := lib.EntryFromPath(cwd); err == nil {
		if filesystemPath, ok := entry.FilesystemPath(); ok && filesystemPath == cwd {
			if abs, err := filepath.Abs(cwd); err == nil {
				cwd = abs
			}
			if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
				cwd = resolved
			}
		}
	}
	return cwd
}

func (m *Model2) restorePlayer() {
	tracks := make([]core.Track, 0, len(m.saved.Player.Playlist))
	for _, entry := range m.saved.Player.Playlist {
		if entry.Path == "" {
			continue
		}
		name := entry.Name
		if name == "" {
			if libraryEntry, err := m.app.Library().EntryFromPath(entry.Path); err == nil {
				name = libraryEntry.Name()
			}
		}
		tracks = append(tracks, core.Track{
			Path: entry.Path, Name: name, Artist: entry.Artist, Title: entry.Title,
			Album: entry.Album, Duration: entry.Duration,
		})
	}
	cursor := m.saved.Player.Cursor
	if cursor < 0 || cursor >= len(tracks) {
		cursor = m.saved.Player.Playing
	}
	m.app.Restore(tracks, cursor, ParseQueueMode(m.saved.Player.QueueMode))
	volume := core.DefaultVolume
	if m.saved.Player.Volume != nil {
		volume = *m.saved.Player.Volume
	}
	m.app.SetVolume(volume)
}

func (m *Model2) openFiles(files []string) {
	if len(files) == 0 {
		return
	}
	startIndex := len(m.app.State().Playlist)
	tracks := make([]core.Track, 0, len(files))
	for _, file := range files {
		entry, err := m.app.Library().EntryFromPath(normalizeInputPath(file))
		if err == nil && entry.IsAudio() {
			tracks = append(tracks, core.Track{Name: entry.Name(), Path: entry.Path()})
		}
	}
	if len(tracks) == 0 {
		return
	}
	_ = m.app.Dispatch(core.Command{Type: core.CmdAddAll, Tracks: tracks})
	_ = m.app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: startIndex})
	_ = m.app.Dispatch(core.Command{Type: core.CmdPlayFromCursor})
}

// SaveState persists the player, lyrics follow setting, and browser directory.
// It retains the original layout's focus and browser visibility settings.
func (m *Model2) SaveState() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	appState := m.app.State()
	tracks := make([]Track, 0, len(appState.Playlist))
	for _, track := range appState.Playlist {
		tracks = append(tracks, Track{
			Path: track.Path, Name: track.Name, Artist: track.Artist, Title: track.Title,
			Album: track.Album, Duration: track.Duration,
		})
	}
	saved := m.saved
	saved.Player = Player{
		Volume: new(appState.Volume), QueueMode: QueueModeString(appState.QueueMode),
		Playlist: tracks, Playing: appState.Playing, Cursor: appState.Cursor,
	}
	saved.Lyrics.FollowLine = m.lyrics.FollowLine()
	saved.Browser.Cwd = m.browser.Cwd
	return Save(path, saved)
}
