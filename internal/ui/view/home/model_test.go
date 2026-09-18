package home

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulatedPanelsFitAssignedSize(t *testing.T) {
	dir := t.TempDir()
	for i := range 30 {
		path := filepath.Join(dir, fmt.Sprintf("track-%02d.mp3", i))
		require.NoError(t, os.WriteFile(path, nil, 0o600))
	}

	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	m := NewModel(Config{
		Cwd:     dir,
		Theme:   theme.Resolve(cfg.TUI.Theme),
		App:     app,
		Library: app.Library(),
		FPS:     cfg.TUI.FPS,
	})
	m.Show(true)
	m.ShowBrowser(true)
	m.FocusBrowser()
	_, _, _ = m.Update(m.browser.Init()())

	const width = 160
	const height = 24
	_, _, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})

	view := m.View()
	lines := strings.Split(view, "\n")

	assert.Equal(t, width, lipgloss.Width(view))
	assert.Equal(t, height, lipgloss.Height(view))
	assert.Contains(t, lines[len(lines)-1], "╰")
	assert.Contains(t, lines[len(lines)-1], "╯")
}
