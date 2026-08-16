package view_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view"
	_ "github.com/bpicode/tmus/testing"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

const testTrack = "Britney Sheers - Maybe One More Line.mp3"

func TestModelNavigatesToHelp(t *testing.T) {
	tui := newTUITest(t)
	tui.waitForHome()

	tui.tm.Type("?")
	tui.waitForOutput("tmus keybindings")

	tui.tm.Type("?")
	tui.waitForOutput("Files", "Playlist")
}

func TestModelQuits(t *testing.T) {
	tui := newTUITest(t)
	tui.waitForHome()

	tui.tm.Type("q")
	tui.waitFinished()
}

func TestModelAddsFileToPlaylist(t *testing.T) {
	tui := newTUITest(t, testTrack)
	tui.waitForHome(testTrack)

	tui.tm.Type("a")
	state := tui.waitForState(func(state core.State) bool {
		return len(state.Playlist) == 1
	})

	if got := state.Playlist[0].Name; got != testTrack {
		t.Fatalf("expected playlist track %q, got %q", testTrack, got)
	}
	if got := state.Playlist[0].Path; got != filepath.Join(tui.startDir, testTrack) {
		t.Fatalf("expected playlist path %q, got %q", filepath.Join(tui.startDir, testTrack), got)
	}

	// The numbered row is unique to the playlist; the browser only renders the filename.
	tui.waitForOutput("1 Britney Sheers - Maybe One More Line")
}

func TestModelRemovesFileFromPlaylist(t *testing.T) {
	tui := newTUITest(t, testTrack)
	tui.waitForHome(testTrack)

	tui.tm.Type("a")
	tui.waitForState(func(state core.State) bool {
		return len(state.Playlist) == 1
	})

	tui.tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.tm.Type("x")
	state := tui.waitForState(func(state core.State) bool {
		return len(state.Playlist) == 0
	})

	if state.Cursor != -1 {
		t.Fatalf("expected no playlist selection, got cursor %d", state.Cursor)
	}
	tui.waitForOutput("(empty)")
}

type tuiTest struct {
	t        *testing.T
	appRef   *core.App
	tm       *teatest.TestModel
	startDir string
}

func newTUITest(t *testing.T, fixtures ...string) *tuiTest {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	startDir := t.TempDir()
	for _, fixture := range fixtures {
		copyFixture(t, startDir, fixture)
	}

	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	cfg.TUI.ArtworkRenderer = "none"
	cfg.TUI.BrowserHome = startDir
	th := theme.Resolve(cfg.TUI.Theme)

	appRef := core.New(cfg)
	t.Cleanup(appRef.ShutdownAndWait)

	tm := teatest.NewTestModel(
		t,
		view.NewModel(appRef, startDir, nil, cfg.TUI, th),
		teatest.WithInitialTermSize(100, 30),
	)
	t.Cleanup(func() {
		_ = tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	tui := &tuiTest{
		t:        t,
		appRef:   appRef,
		tm:       tm,
		startDir: startDir,
	}
	return tui
}

func (tui *tuiTest) waitForHome(values ...string) {
	tui.t.Helper()
	tui.waitForOutput(append([]string{"Files", "Playlist"}, values...)...)
}

func (tui *tuiTest) waitForOutput(values ...string) {
	tui.t.Helper()
	teatest.WaitFor(
		tui.t,
		tui.tm.Output(),
		func(output []byte) bool {
			for _, value := range values {
				if !bytes.Contains(output, []byte(value)) {
					return false
				}
			}
			return true
		},
		teatest.WithDuration(2*time.Second),
		teatest.WithCheckInterval(10*time.Millisecond),
	)
}

func (tui *tuiTest) waitForState(condition func(core.State) bool) core.State {
	tui.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := tui.appRef.State()
		if condition(state) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	state := tui.appRef.State()
	tui.t.Fatalf("condition not met after 2s; last application state: %+v", state)
	return state
}

func (tui *tuiTest) waitFinished() {
	tui.t.Helper()
	tui.tm.WaitFinished(tui.t, teatest.WithFinalTimeout(time.Second))
}

func copyFixture(t *testing.T, targetDir, name string) {
	t.Helper()
	source, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer source.Close()

	target, err := os.OpenFile(filepath.Join(targetDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatalf("create fixture copy: %v", err)
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		t.Fatalf("copy fixture: %v", err)
	}
	if err := target.Close(); err != nil {
		t.Fatalf("close fixture copy: %v", err)
	}
}
