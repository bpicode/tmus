package tabs_test

import (
	"testing"

	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		items  []tabs.Tab
		active tabs.Tab
		ok     bool
	}{
		{name: "nil"},
		{name: "empty", items: []tabs.Tab{}},
		{
			name:   "single tab",
			items:  []tabs.Tab{{ID: "one", Label: "One"}},
			active: tabs.Tab{ID: "one", Label: "One"},
			ok:     true,
		},
		{
			name:   "first tab selected",
			items:  []tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}},
			active: tabs.Tab{ID: "one", Label: "One"},
			ok:     true,
		},
		{
			name:   "shared labels",
			items:  []tabs.Tab{{ID: "one", Label: "Same"}, {ID: "two", Label: "Same"}},
			active: tabs.Tab{ID: "one", Label: "Same"},
			ok:     true,
		},
		{
			name:   "empty label",
			items:  []tabs.Tab{{ID: "one"}},
			active: tabs.Tab{ID: "one"},
			ok:     true,
		},
		{
			name:   "unicode",
			items:  []tabs.Tab{{ID: "音楽", Label: "音楽 🎵"}},
			active: tabs.Tab{ID: "音楽", Label: "音楽 🎵"},
			ok:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New(tt.items)
			require.NoError(t, err)
			active, ok := m.Active()
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.active, active)
			assert.Equal(t, tt.active.ID, m.ActiveID())
		})
	}
}

func TestNewInvalidIDs(t *testing.T) {
	tests := []struct {
		name  string
		items []tabs.Tab
		err   string
	}{
		{
			name:  "empty first ID",
			items: []tabs.Tab{{Label: "One"}},
			err:   "tab at index 0 has an empty ID",
		},
		{
			name:  "empty later ID",
			items: []tabs.Tab{{ID: "one"}, {Label: "Two"}},
			err:   "tab at index 1 has an empty ID",
		},
		{
			name:  "duplicate ID",
			items: []tabs.Tab{{ID: "one", Label: "One"}, {ID: "one", Label: "Two"}},
			err:   `duplicate tab ID "one"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New(tt.items)
			require.ErrorContains(t, err, tt.err)
			active, ok := m.Active()
			assert.False(t, ok)
			assert.Equal(t, tabs.Tab{}, active)
			assert.Empty(t, m.ActiveID())
		})
	}
}

func TestSelect(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		id      string
		want    tabs.Tab
		wantErr bool
	}{
		{
			name: "select another tab", initial: "one", id: "two",
			want: tabs.Tab{ID: "two", Label: "Two"},
		},
		{
			name: "return to first tab", initial: "two", id: "one",
			want: tabs.Tab{ID: "one", Label: "One"},
		},
		{
			name: "select active tab", initial: "two", id: "two",
			want: tabs.Tab{ID: "two", Label: "Two"},
		},
		{
			name: "unknown ID", initial: "two", id: "missing",
			want: tabs.Tab{ID: "two", Label: "Two"}, wantErr: true,
		},
		{
			name: "empty ID", initial: "two", id: "",
			want: tabs.Tab{ID: "two", Label: "Two"}, wantErr: true,
		},
		{
			name: "label is not an ID", initial: "two", id: "One",
			want: tabs.Tab{ID: "two", Label: "Two"}, wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}})
			require.NoError(t, err)
			require.NoError(t, m.Select(tt.initial))
			err = m.Select(tt.id)
			if tt.wantErr {
				require.ErrorContains(t, err, "unknown tab ID")
			} else {
				require.NoError(t, err)
			}
			active, ok := m.Active()
			assert.True(t, ok)
			assert.Equal(t, tt.want, active)
			assert.Equal(t, tt.want.ID, m.ActiveID())
		})
	}
}

func TestNewCopiesTabs(t *testing.T) {
	items := []tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}}
	m, err := tabs.New(items)
	require.NoError(t, err)

	items[0] = tabs.Tab{ID: "changed", Label: "Changed"}
	items[1] = tabs.Tab{ID: "also-changed", Label: "Also changed"}
	active, ok := m.Active()
	assert.True(t, ok)
	assert.Equal(t, tabs.Tab{ID: "one", Label: "One"}, active)
	require.NoError(t, m.Select("two"))
	active, ok = m.Active()
	assert.True(t, ok)
	assert.Equal(t, tabs.Tab{ID: "two", Label: "Two"}, active)

	active.ID = "changed"
	active.Label = "Changed"
	active, _ = m.Active()
	assert.Equal(t, tabs.Tab{ID: "two", Label: "Two"}, active)
}

func TestModelCopySelection(t *testing.T) {
	m, err := tabs.New([]tabs.Tab{{ID: "one"}, {ID: "two"}})
	require.NoError(t, err)
	other := m
	require.NoError(t, other.Select("two"))
	assert.Equal(t, "one", m.ActiveID())
	assert.Equal(t, "two", other.ActiveID())
}

func TestZeroValue(t *testing.T) {
	var m tabs.Model
	active, ok := m.Active()
	assert.False(t, ok)
	assert.Equal(t, tabs.Tab{}, active)
	assert.Empty(t, m.ActiveID())
	require.ErrorContains(t, m.Select("one"), "unknown tab ID")
	assert.Empty(t, m.ActiveID())
}
