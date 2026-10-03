package view_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
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
		{name: "next", key: tea.KeyPressMsg{Code: tea.KeyTab}, id: "track", text: "Track"},
		{name: "previous wraps", key: tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, id: "help", text: "Help"},
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
			assert.Contains(t, updated.View().Content, tt.text+" content will be added next.")
			_, cmd = m.Update(change)
			assert.Nil(t, cmd)
			assert.Contains(t, m.View().Content, tt.text+" content will be added next.")
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
	tui.waitForOutput("Track content will be added next.")
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
	_, open := <-metadata
	assert.False(t, open, "app shutdown finishes event forwarding")
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
	assert.Contains(t, m.View().Content, "Track content will be added next.")
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
