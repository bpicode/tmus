package volume

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestModelView(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		wantLabel bool
		wantWidth int
	}{
		{name: "no width", width: 0},
		{name: "label fills view", width: 8},
		{name: "room for label and bar", width: 24, wantLabel: true, wantWidth: 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel("Volume: ", testTheme())
			m.UpdateSize(tt.width)

			view := m.View(50)

			if tt.wantLabel {
				assert.Contains(t, view, "Volume: ")
			} else {
				assert.NotContains(t, view, "Volume: ")
			}
			assert.Contains(t, view, "50%")
			if tt.wantWidth > 0 {
				assert.Equal(t, tt.wantWidth, lipgloss.Width(view))
			}
		})
	}
}

func testTheme() theme.Theme {
	return theme.Theme{
		Primary:   lipgloss.Color("#ffffff"),
		Secondary: lipgloss.Color("#000000"),
	}
}
