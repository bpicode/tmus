package view_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	applyrics "github.com/bpicode/tmus/internal/app/lyrics"
	"github.com/bpicode/tmus/internal/ui/view"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelReceivesLyricsEvents(t *testing.T) {
	dir := t.TempDir()
	tracks := []core.Track{
		{Path: filepath.Join(dir, "one.mp3"), Name: "First song"},
		{Path: filepath.Join(dir, "two.mp3"), Name: "Second song"},
	}
	for _, track := range tracks {
		// Queue these files without playing them; only the sidecar is read.
		require.NoError(t, os.WriteFile(track.Path, nil, 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.lrc"), []byte("[00:01.00]First sidecar line"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "two.lrc"), []byte("[00:01.00]Second sidecar line"), 0o600))
	m, app := newModelTest(t)
	app.Restore(tracks, 0, core.QueueModeLinear)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
	tui := &tuiTest{t: t, appRef: app, tm: tm}
	tui.waitForOutput("Search: /", "First song")
	tm.Type("L")
	tui.waitForOutput("First sidecar line")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEscape})
	tui.waitForOutput("Search: /")
	require.NoError(t, app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: 1}))
	tui.waitForState(func(state core.State) bool { return state.Cursor == 1 })
	tm.Type("L")
	tui.waitForOutput("Second sidecar line")
	tm.Type("q")
	tui.waitFinished()
}

func TestModelLyricsKeys(t *testing.T) {
	for _, press := range []tea.KeyPressMsg{
		{Code: tea.KeyEscape}, {Code: 'L', Text: "L"},
		{Code: 'q', Text: "q"}, {Code: 'c', Mod: tea.ModCtrl},
	} {
		t.Run(press.String(), func(t *testing.T) {
			m, app := newModelTest(t)
			app.Restore([]core.Track{{Path: "/music/one.flac", Name: "One"}}, 0, core.QueueModeLinear)
			_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			_, cmd := m.Update(tea.KeyPressMsg{Code: 'L', Text: "L"})
			require.NotNil(t, cmd)
			_, _ = m.Update(cmd())
			assert.Contains(t, m.View().Content, "Loading...")
			_, cmd = m.Update(press)
			if press.String() == "q" || press.String() == "ctrl+c" {
				require.NotNil(t, cmd)
				assert.IsType(t, tea.QuitMsg{}, cmd())
			} else {
				assert.Contains(t, m.View().Content, "Search: /")
				assert.NotContains(t, m.View().Content, "Loading...")
			}
		})
	}
}

func TestModelLyricsScrollAndFollowSetting(t *testing.T) {
	m, app := newModelTest(t)
	app.Restore([]core.Track{{Path: "/music/one.flac", Name: "One"}}, 0, core.QueueModeLinear)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	track := app.State().Playlist[0]
	_, _ = m.Update(core.LyricsEvent{
		TrackID: track.ID, Path: track.Path,
		Lyrics: applyrics.Lyrics{Lines: []applyrics.Line{
			{Text: "First lyric"}, {Text: "Second lyric"}, {Text: "Third lyric"},
			{Text: "Fourth lyric"}, {Text: "Fifth lyric"}, {Text: "Sixth lyric"},
			{Text: "Seventh lyric"}, {Text: "Eighth lyric"}, {Text: "Last lyric"},
		}},
	})
	assert.Contains(t, m.View().Content, "First lyric")
	assert.NotContains(t, m.View().Content, "Last lyric")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	assert.Contains(t, m.View().Content, "Last lyric")
	assert.NotContains(t, m.View().Content, "First lyric")
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlayback})
	assert.Contains(t, m.View().Content, "Last lyric", "background updates preserve manual scrolling")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	assert.Contains(t, m.View().Content, "First lyric")

	_, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	require.NoError(t, m.SaveState())
	path, err := view.DefaultPath()
	require.NoError(t, err)
	saved, err := view.Load(path)
	require.NoError(t, err)
	assert.True(t, saved.Lyrics.FollowLine, "f changes the setting only while Lyrics is active")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	require.NoError(t, m.SaveState())
	saved, err = view.Load(path)
	require.NoError(t, err)
	assert.False(t, saved.Lyrics.FollowLine)
}

func TestModelLyricsEmptyAndResize(t *testing.T) {
	m, app := newModelTest(t)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	assert.Contains(t, m.View().Content, "No track selected.")
	app.Restore([]core.Track{{Path: "/music/one.flac", Name: "One"}}, 0, core.QueueModeLinear)
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	assert.Contains(t, m.View().Content, "Loading...")
	track := app.State().Playlist[0]
	_, _ = m.Update(core.LyricsEvent{TrackID: track.ID, Path: track.Path})
	assert.Contains(t, m.View().Content, "No lyrics available")
	_, _ = m.Update(core.LyricsEvent{
		TrackID: track.ID + 1, Path: track.Path,
		Lyrics: applyrics.Lyrics{Lines: []applyrics.Line{{Text: "Stale lyric"}}},
	})
	assert.NotContains(t, m.View().Content, "Stale lyric")
	for _, size := range []tea.WindowSizeMsg{{Width: 70, Height: 18}, {Width: 90, Height: 30}} {
		_, _ = m.Update(size)
		content := m.View().Content
		assert.Equal(t, size.Width, lipgloss.Width(content))
		assert.Equal(t, size.Height, lipgloss.Height(content))
		assert.Contains(t, content, "No lyrics available")
	}
	app.Restore(nil, -1, core.QueueModeLinear)
	_, _ = m.Update(core.StateEvent{Changes: core.StateChangePlaylist})
	assert.Contains(t, m.View().Content, "No track selected.")
	assert.NotContains(t, m.View().Content, "No lyrics available")
}
