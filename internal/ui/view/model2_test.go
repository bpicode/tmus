package view_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel2Navigation(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
		id   string
		text string
	}{
		{name: "next", key: tea.KeyPressMsg{Code: tea.KeyTab}, id: "track", text: "No track selected."},
		{name: "previous wraps", key: tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, id: "help", text: "Help content will be added next."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := newModel2Test(t)
			_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			assert.Contains(t, m.View().Content, "(empty)")
			updated, cmd := m.Update(tt.key)
			require.NotNil(t, cmd)
			change := cmd()
			assert.Equal(t, tabs.ChangeMsg{Previous: "playlist", Current: tt.id}, change)
			assert.Contains(t, updated.View().Content, tt.text)
			_, cmd = m.Update(change)
			assert.Nil(t, cmd)
			assert.Contains(t, m.View().Content, tt.text)
		})
	}
}

func TestModel2ResizesFrame(t *testing.T) {
	th := theme.Resolve(config.Default().TUI.Theme)
	m, _ := newModel2Test(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 60, Height: 12}} {
		_, cmd := m.Update(size)
		assert.Nil(t, cmd)
		v := m.View()
		assert.Equal(t, size.Width, lipgloss.Width(v.Content))
		assert.Equal(t, size.Height, lipgloss.Height(v.Content))
		assert.Contains(t, ansi.Strip(v.Content), "│ Playlist │ Track │ Lyrics │ Browser │ Help │")
		assert.True(t, v.AltScreen)
		assert.Equal(t, th.Foreground, v.ForegroundColor)
		assert.Equal(t, th.Background, v.BackgroundColor)
	}
}

func TestModel2Quits(t *testing.T) {
	for _, press := range []tea.KeyPressMsg{{Code: 'q'}, {Code: 'c', Mod: tea.ModCtrl}} {
		t.Run(press.String(), func(t *testing.T) {
			m, app := newModel2Test(t)
			_, cmd := m.Update(press)
			require.NotNil(t, cmd)
			assert.IsType(t, tea.QuitMsg{}, cmd())
			require.NoError(t, app.Dispatch(core.Command{Type: core.CmdSetVolume, Volume: 37}), "the runner owns app shutdown")
		})
	}
}

func newModel2Test(t *testing.T, files ...string) (*view.Model2, *core.App) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	m, err := view.NewModel2(app, files, cfg.TUI, theme.Resolve(cfg.TUI.Theme))
	require.NoError(t, err)
	t.Cleanup(m.Shutdown)
	return m, app
}

func TestModel2ReceivesStateEvents(t *testing.T) {
	m, app := newModel2Test(t)
	metadata, unsubscribe := app.SubscribeMetadataEvents()
	t.Cleanup(unsubscribe)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
	tui := &tuiTest{t: t, appRef: app, tm: tm}
	tui.waitForOutput("(empty)")
	require.NoError(t, app.Dispatch(core.Command{Type: core.CmdAddAll,
		Tracks: []core.Track{{Path: "/music/one.flac", Name: "First background track"}}}))
	tui.waitForOutput("First background track")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.waitForOutput("Track info")
	require.NoError(t, app.Dispatch(core.Command{Type: core.CmdAddAll,
		Tracks: []core.Track{{Path: "/music/two.flac", Name: "Second background track"}}}))
	tui.waitForState(func(state core.State) bool { return len(state.Playlist) == 2 })
	tm.Send(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	tui.waitForOutput("Second background track")
	tm.Type("q")
	tui.waitFinished()
	m.Shutdown()
	require.NoError(t, app.Dispatch(core.Command{Type: core.CmdSetVolume, Volume: 37}), "model cleanup leaves the app running")
	app.ShutdownAndWait()
	for {
		select {
		case _, open := <-metadata:
			if !open {
				return
			}
		default:
			t.Fatal("metadata events are still open after app shutdown")
		}
	}
}

func TestModel2OpensAudioArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "example.mp3")
	// An empty file exercises queue insertion without producing audio.
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	_, app := newModel2Test(t, path, filepath.Join(t.TempDir(), "notes.txt"))
	require.Eventually(t, func() bool {
		state := app.State()
		return len(state.Playlist) == 1
	}, time.Second, time.Millisecond)
	assert.Equal(t, path, app.State().Playlist[0].Path)
}

func TestModel2UpdatesPlaylistWhileHidden(t *testing.T) {
	m, app := newModel2Test(t)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	app.Restore([]core.Track{{Path: "/music/example.flac", Name: "Updated track"}}, 0, core.QueueModeLinear)
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	assert.Contains(t, m.View().Content, "Track info")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	assert.Contains(t, m.View().Content, "Updated track")
}

func TestModel2RoutesPlaylistKeys(t *testing.T) {
	m, app := newModel2Test(t)
	tracks := []core.Track{{Path: "/music/one.flac", Name: "One"}, {Path: "/music/two.flac", Name: "Two"}}
	app.Restore(tracks, 0, core.QueueModeLinear)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 0, app.State().Cursor, "inactive playlist ignores navigation")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Eventually(t, func() bool { return app.State().Cursor == 1 }, time.Second, time.Millisecond)
}

func TestModel2SearchAcceptsQuitLetter(t *testing.T) {
	m, app := newModel2Test(t)
	app.Restore([]core.Track{{Path: "/music/quiet.flac", Name: "Quiet track"}}, 0, core.QueueModeLinear)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	_, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	assert.Contains(t, ansi.Strip(m.View().Content), "q", "q is entered into the search")
	require.NoError(t, app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: 0}), "search does not shut down the app")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}

func TestModel2RestoresAndSavesPlayerState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := view.DefaultPath()
	require.NoError(t, err)
	saved := view.State{
		Focus: "browser", Browser: view.Browser{Cwd: filepath.Join(t.TempDir(), "music"), Hidden: true},
		Lyrics: view.Lyrics{FollowLine: true},
		Player: view.Player{
			Volume: new(37), QueueMode: "repeat-all", Playing: -1, Cursor: 0,
			Playlist: []view.Track{{Path: "/music/one.flac", Name: "Restored track"}},
		},
	}
	require.NoError(t, view.Save(path, saved))
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	m, err := view.NewModel2(app, nil, cfg.TUI, theme.Resolve(cfg.TUI.Theme))
	require.NoError(t, err)
	t.Cleanup(m.Shutdown)
	assert.Equal(t, "Restored track", app.State().Playlist[0].Name)
	assert.Equal(t, 37, app.State().Volume)
	assert.Equal(t, core.QueueModeRepeatAll, app.State().QueueMode)
	app.SetVolume(42)
	require.NoError(t, m.SaveState())
	got, err := view.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 42, *got.Player.Volume)
	assert.Equal(t, saved.Player.Playlist, got.Player.Playlist)
	assert.Equal(t, saved.Browser, got.Browser)
	assert.Equal(t, saved.Lyrics, got.Lyrics)
	assert.Equal(t, saved.Focus, got.Focus)
}

func TestModel2TrackFollowsSelection(t *testing.T) {
	m, app := newModel2Test(t)
	app.Restore([]core.Track{
		{Path: "/music/one.flac", Name: "One"},
		{Path: "/music/two.flac", Name: "Two"},
	}, 0, core.QueueModeLinear)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	require.NotNil(t, cmd)
	_, _ = m.Update(cmd())
	assert.Contains(t, m.View().Content, "Track info")
	assert.Contains(t, m.View().Content, "/music/one.flac")
	assert.Contains(t, m.View().Content, "Loading...")
	tracks := app.State().Playlist
	first := core.MetadataEvent{
		TrackID: tracks[0].ID, Path: tracks[0].Path, Scope: core.MetadataExtended,
		Metadata: library.Metadata{Title: "First title", Artist: "First artist"},
	}
	_, _ = m.Update(first)
	assert.Contains(t, m.View().Content, "First title")
	assert.Contains(t, m.View().Content, "First artist")
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangeVolume})
	assert.Contains(t, m.View().Content, "First title", "unrelated updates retain loaded metadata")

	require.NoError(t, app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: 1}))
	require.Eventually(t, func() bool { return app.State().Cursor == 1 }, time.Second, time.Millisecond)
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangeSelection})
	assert.Contains(t, m.View().Content, "/music/two.flac")
	assert.NotContains(t, m.View().Content, "First title")
	_, _ = m.Update(first)
	assert.NotContains(t, m.View().Content, "First title", "late metadata for the previous track is ignored")
	_, _ = m.Update(core.MetadataEvent{
		TrackID: tracks[1].ID, Path: tracks[1].Path, Scope: core.MetadataExtended,
		Metadata: library.Metadata{Title: "Second title"},
	})
	assert.Contains(t, m.View().Content, "Second title")

	for _, size := range []tea.WindowSizeMsg{{Width: 70, Height: 18}, {Width: 90, Height: 30}} {
		_, _ = m.Update(size)
		content := m.View().Content
		assert.Equal(t, size.Width, lipgloss.Width(content))
		assert.Equal(t, size.Height, lipgloss.Height(content))
		assert.Contains(t, content, "Second title")
	}
	app.Restore(nil, -1, core.QueueModeLinear)
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	assert.Contains(t, m.View().Content, "No track selected.")
	assert.NotContains(t, m.View().Content, "Second title")
}

func TestModel2TrackKeys(t *testing.T) {
	for _, press := range []tea.KeyPressMsg{
		{Code: tea.KeyEscape}, {Code: 'i', Text: "i"},
		{Code: 'q', Text: "q"}, {Code: 'c', Mod: tea.ModCtrl},
	} {
		t.Run(press.String(), func(t *testing.T) {
			m, app := newModel2Test(t)
			app.Restore([]core.Track{{Path: "/music/one.flac", Name: "One"}}, 0, core.QueueModeLinear)
			_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			_, cmd := m.Update(press)
			if press.String() == "q" || press.String() == "ctrl+c" {
				require.NotNil(t, cmd)
				assert.IsType(t, tea.QuitMsg{}, cmd())
			} else {
				assert.Contains(t, m.View().Content, "Search: /")
				assert.NotContains(t, m.View().Content, "Track info")
			}
		})
	}
}

func TestModel2TrackScrolls(t *testing.T) {
	m, app := newModel2Test(t)
	app.Restore([]core.Track{{Path: "/music/one.flac", Name: "One"}}, 0, core.QueueModeLinear)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 14})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	track := app.State().Playlist[0]
	_, _ = m.Update(core.MetadataEvent{
		TrackID: track.ID, Path: track.Path, Scope: core.MetadataExtended,
		Metadata: library.Metadata{Title: "At the top", Year: 2026},
	})
	assert.Contains(t, m.View().Content, "At the top")
	assert.NotContains(t, m.View().Content, "2026")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	assert.Contains(t, m.View().Content, "2026")
	assert.NotContains(t, m.View().Content, "At the top")
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangeVolume})
	assert.Contains(t, m.View().Content, "2026", "unrelated state updates retain scroll position")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	assert.Contains(t, m.View().Content, "At the top")
}

func TestModel2ReleasesTrackArtworkOnTabChange(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	m, app := newModel2Test(t)
	app.Restore([]core.Track{{Path: "/music/one.flac", Name: "One"}}, 0, core.QueueModeLinear)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	var picture bytes.Buffer
	require.NoError(t, png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	track := app.State().Playlist[0]
	_, cmd := m.Update(core.MetadataEvent{
		TrackID: track.ID, Path: track.Path, Scope: core.MetadataExtended,
		Metadata: library.Metadata{Picture: &library.Picture{Data: picture.Bytes()}},
	})
	require.NotNil(t, cmd)
	// The other command in this batch waits for the next metadata event.
	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	upload, ok := batch[len(batch)-1]().(tea.RawMsg)
	require.True(t, ok)
	assert.Contains(t, upload.Msg, "a=T", "the configured artwork renderer uploads the picture")
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	require.NotNil(t, cmd)
	batch, ok = cmd().(tea.BatchMsg)
	require.True(t, ok)
	cleanup, ok := batch[len(batch)-1]().(tea.RawMsg)
	require.True(t, ok)
	assert.Contains(t, cleanup.Msg, "a=d", "leaving Track deletes its terminal image")
	assert.Contains(t, m.View().Content, "Lyrics content will be added next.")
}
