package track_info_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/ui/view/track_info"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowReturnsMetadataRequests(t *testing.T) {
	m := track_info.NewModel(track_info.Config{})
	_, _, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	state := core.State{
		Playlist: []core.Track{
			{ID: 1, Path: "/music/one.flac"},
			{ID: 2, Path: "/music/two.flac"},
		},
		Cursor: 0, Playing: -1,
	}
	_, request := m.Show(state)
	require.Equal(t, &track_info.MetadataRequest{TrackID: 1, Path: "/music/one.flac"}, request)
	assert.Contains(t, m.View(), "Loading...")
	metadata := core.MetadataEvent{
		TrackID: request.TrackID, Path: request.Path, Scope: core.MetadataExtended,
		Metadata: library.Metadata{Title: "Loaded title"},
	}
	_, _, _ = m.Update(metadata)
	assert.Contains(t, ansi.Strip(m.View()), "Loaded title")

	cmd, request := m.Show(state)
	assert.Nil(t, cmd)
	assert.Nil(t, request, "refreshing the same track does not request metadata again")
	assert.Contains(t, ansi.Strip(m.View()), "Loaded title")

	state.Cursor = 1
	_, request = m.Show(state)
	assert.Equal(t, &track_info.MetadataRequest{TrackID: 2, Path: "/music/two.flac"}, request)
	_, _, _ = m.Update(metadata)
	assert.Contains(t, m.View(), "Loading...")
	assert.NotContains(t, ansi.Strip(m.View()), "Loaded title", "a late reply belongs to the previous track")

	m.Hide()
	_, request = m.Show(state)
	assert.Equal(t, &track_info.MetadataRequest{TrackID: 2, Path: "/music/two.flac"}, request,
		"reopening a track requests its metadata")
}

func TestShowWithoutTrackReturnsNoRequest(t *testing.T) {
	tests := []struct {
		name  string
		state core.State
	}{
		{name: "empty playlist"},
		{name: "invalid selection", state: core.State{Playlist: []core.Track{{ID: 1, Path: "/music/one.flac"}}, Cursor: 2}},
		{name: "missing identity", state: core.State{Playlist: []core.Track{{Path: "/music/one.flac"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := track_info.NewModel(track_info.Config{})
			_, request := m.Show(tt.state)
			assert.Nil(t, request)
			assert.False(t, m.Visible())
		})
	}
}
