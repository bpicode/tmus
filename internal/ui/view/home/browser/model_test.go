package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowSizeUpdatesListLayout(t *testing.T) {
	m := NewModel(Config{Cwd: t.TempDir()})

	const width = 80
	const height = 10
	_, _, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})

	innerWidth := width - m.styles.panelUnfocused.GetHorizontalFrameSize()
	innerHeight := height - m.styles.panelUnfocused.GetVerticalFrameSize()
	assert.Equal(t, innerWidth, m.layout.innerWidth)
	assert.Equal(t, innerHeight-headerHeight, m.layout.bodyHeight)
	assert.Equal(t, m.layout.innerWidth, m.list.Width())
	assert.Equal(t, m.layout.bodyHeight, m.list.Height())
}

func TestWindowSizeClampsListLayout(t *testing.T) {
	m := NewModel(Config{Cwd: t.TempDir()})

	_, _, _ = m.Update(tea.WindowSizeMsg{Width: 1, Height: 1})

	assert.Zero(t, m.layout.innerWidth)
	assert.Zero(t, m.layout.bodyHeight)
	assert.Zero(t, m.list.Width())
	assert.Zero(t, m.list.Height())
}

func TestPopulatedViewFitsAssignedSize(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(Config{Cwd: dir})
	for i := range 30 {
		path := filepath.Join(dir, fmt.Sprintf("track-%02d.mp3", i))
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		entry, err := m.lib.EntryFromPath(path)
		require.NoError(t, err)
		m.entries = append(m.entries, entry)
	}
	m.updateListItems(0)

	const width = 80
	for height := 7; height <= 40; height++ {
		t.Run(fmt.Sprintf("height %d", height), func(t *testing.T) {
			_, _, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})

			view := m.View()

			assert.Equal(t, width, lipgloss.Width(view))
			assert.Equal(t, height, lipgloss.Height(view))
		})
	}
}

func TestLoadingEntriesSettlesPagination(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(Config{Cwd: dir})
	_, _, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	for i := range 19 {
		path := filepath.Join(dir, fmt.Sprintf("track-%02d.mp3", i))
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		entry, err := m.lib.EntryFromPath(path)
		require.NoError(t, err)
		m.entries = append(m.entries, entry)
	}
	m.updateListItems(0)

	assert.Greater(t, m.list.Paginator.TotalPages, 1)
	assert.Contains(t, m.View(), "•")
}
