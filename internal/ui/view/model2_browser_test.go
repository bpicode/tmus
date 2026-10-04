package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBrowserModel2Test(t *testing.T, startDir string, saved view.State) (*view.Model2, *core.App) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := view.DefaultPath()
	require.NoError(t, err)
	require.NoError(t, view.Save(path, saved))
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	cfg.TUI.BrowserHome = startDir
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	m, err := view.NewModel2(app, startDir, nil, cfg.TUI, theme.Resolve(cfg.TUI.Theme))
	require.NoError(t, err)
	t.Cleanup(m.Shutdown)
	return m, app
}

func startBrowserModel2Test(t *testing.T, m tea.Model, app *core.App, entries ...string) *tuiTest {
	t.Helper()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
	tui := &tuiTest{t: t, appRef: app, tm: tm}
	tui.waitForOutput("Search: /")
	tm.Type("b")
	tui.waitForOutput(append([]string{"Files"}, entries...)...)
	return tui
}

func TestModel2BrowserStartingDirectory(t *testing.T) {
	explicitDir, savedDir := t.TempDir(), t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	for _, tt := range []struct {
		name, start, saved, want string
	}{
		{name: "explicit overrides saved", start: explicitDir, saved: savedDir, want: explicitDir},
		{name: "saved directory", saved: savedDir, want: savedDir},
		{name: "working directory", want: wd},
		{name: "relative directory", start: ".", want: wd},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := newBrowserModel2Test(t, tt.start, view.State{
				Focus: "browser", Browser: view.Browser{Cwd: tt.saved, Hidden: true},
			})
			require.NoError(t, m.SaveState())
			path, err := view.DefaultPath()
			require.NoError(t, err)
			saved, err := view.Load(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, saved.Browser.Cwd)
			assert.True(t, saved.Browser.Hidden, "the original layout's visibility is retained")
			assert.Equal(t, "browser", saved.Focus)
			for range 3 {
				_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			}
			for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 60, Height: 12}} {
				_, _ = m.Update(size)
				content := m.View().Content
				assert.Contains(t, content, "Files")
				assert.Equal(t, size.Width, lipgloss.Width(content))
				assert.Equal(t, size.Height, lipgloss.Height(content))
			}
		})
	}
}

func TestModel2BrowserAddsTracks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"one.mp3", "two.flac"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0o600))
	}
	m, app := newBrowserModel2Test(t, dir, view.State{})
	tui := startBrowserModel2Test(t, m, app, "one.mp3", "two.flac")
	tui.tm.Type("a")
	state := tui.waitForState(func(state core.State) bool { return len(state.Playlist) == 1 })
	assert.Equal(t, filepath.Join(dir, "one.mp3"), state.Playlist[0].Path)
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	state = tui.waitForState(func(state core.State) bool { return len(state.Playlist) == 2 })
	assert.Equal(t, filepath.Join(dir, "two.flac"), state.Playlist[1].Path)
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.waitForOutput("1 one.mp3", "2 two.flac")
	// Browser actions are inactive while Playlist has focus.
	tui.tm.Type("a")
	tui.tm.Type("q")
	tui.waitFinished()
	assert.Len(t, app.State().Playlist, 2)
}

func TestModel2BrowserNavigatesAndSavesDirectory(t *testing.T) {
	dir := t.TempDir()
	album := filepath.Join(dir, "album")
	require.NoError(t, os.Mkdir(album, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(album, "song.mp3"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden.mp3"), nil, 0o600))
	m, app := newBrowserModel2Test(t, dir, view.State{})
	tui := startBrowserModel2Test(t, m, app, "album")
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Directory replies must still reach Browser while another tab is active.
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	tui.waitForOutput("No track selected.")
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.waitForOutput("song.mp3")
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyBackspace})
	tui.waitForOutput("album")
	tui.tm.Type("H")
	tui.waitForOutput(".hidden.mp3")
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tui.waitForOutput("song.mp3")
	tui.tm.Type("~")
	tui.waitForOutput("album", ".hidden.mp3")
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tui.waitForOutput("song.mp3")
	tui.tm.Type("q")
	tui.waitFinished()
	require.NoError(t, m.SaveState())
	path, err := view.DefaultPath()
	require.NoError(t, err)
	saved, err := view.Load(path)
	require.NoError(t, err)
	assert.Equal(t, album, saved.Browser.Cwd)
}

func TestModel2BrowserAndPlaylistSearchesStaySeparate(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"quiet.mp3", "other.mp3"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0o600))
	}
	m, app := newBrowserModel2Test(t, dir, view.State{})
	app.Restore([]core.Track{{Path: "/music/playlist.flac", Name: "Playlist entry"}}, 0, core.QueueModeLinear)
	observed := &observedModel2{Model: m}
	tui := startBrowserModel2Test(t, observed, app, "quiet.mp3", "other.mp3")
	tui.tm.Type("/q")
	observed.waitForView(t, func(content string) bool {
		return strings.Contains(content, "Search: q") && strings.Contains(content, "quiet.mp3") && !strings.Contains(content, "other.mp3")
	})
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tui.tm.Type("A")
	state := tui.waitForState(func(state core.State) bool { return len(state.Playlist) == 2 })
	assert.Equal(t, filepath.Join(dir, "quiet.mp3"), state.Playlist[1].Path)
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.waitForOutput("Playlist entry")
	tui.tm.Type("/playlist")
	observed.waitForView(t, func(content string) bool {
		return strings.Contains(content, "Search: playlist") && strings.Contains(content, "Playlist entry") && !strings.Contains(content, "quiet.mp3")
	})
	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	for range 3 {
		tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	observed.waitForView(t, func(content string) bool {
		return strings.Contains(content, "Files") && strings.Contains(content, "Search: q") && strings.Contains(content, "quiet.mp3") && !strings.Contains(content, "other.mp3")
	})
	tui.tm.Type("A")
	state = tui.waitForState(func(state core.State) bool { return len(state.Playlist) == 3 })
	assert.Equal(t, filepath.Join(dir, "quiet.mp3"), state.Playlist[2].Path)
	tui.tm.Type("q")
	tui.waitFinished()
}

// Observe complete views on the program goroutine rather than reconstructing
// the renderer's incremental terminal output or reading the model concurrently.
type observedModel2 struct {
	tea.Model
	mu      sync.Mutex
	content string
}

func (m *observedModel2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

func (m *observedModel2) View() tea.View {
	v := m.Model.View()
	m.mu.Lock()
	m.content = ansi.Strip(v.Content)
	m.mu.Unlock()
	return v
}

func (m *observedModel2) waitForView(t *testing.T, matches func(string) bool) {
	t.Helper()
	require.Eventually(t, func() bool {
		m.mu.Lock()
		content := m.content
		m.mu.Unlock()
		return matches(content)
	}, 2*time.Second, 10*time.Millisecond)
}
