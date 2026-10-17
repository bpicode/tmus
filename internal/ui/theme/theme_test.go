package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestResolveConstructsTheme(t *testing.T) {
	tests := []struct {
		name   string
		border string
		want   lipgloss.Border
	}{
		{name: "preset default", want: lipgloss.RoundedBorder()},
		{name: "rounded", border: "rounded", want: lipgloss.RoundedBorder()},
		{name: "normal", border: "normal", want: lipgloss.NormalBorder()},
		{name: "ascii", border: "ascii", want: lipgloss.ASCIIBorder()},
		{name: "block", border: "block", want: lipgloss.BlockBorder()},
		{name: "double", border: "double", want: lipgloss.DoubleBorder()},
		{name: "thick", border: "thick", want: lipgloss.ThickBorder()},
		{name: "hidden", border: "hidden", want: lipgloss.HiddenBorder()},
		{name: "markdown", border: "markdown", want: lipgloss.MarkdownBorder()},
		{name: "unknown falls back", border: "unknown", want: lipgloss.RoundedBorder()},
		{name: "none falls back", border: "none", want: lipgloss.RoundedBorder()},
		{name: "inner half block falls back", border: "inner-half-block", want: lipgloss.RoundedBorder()},
		{name: "outer half block falls back", border: "outer-half-block", want: lipgloss.RoundedBorder()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := Resolve(config.ThemeConfig{
				Preset:     "matrix",
				Primary:    "#abcdef",
				Background: terminalColor,
				Border:     tt.border,
			})

			assert.Equal(t, lipgloss.Color("#abcdef"), th.Primary)
			assert.Equal(t, lipgloss.Color("#c1ff8a"), th.Secondary)
			assert.Nil(t, th.Background)
			assert.Equal(t, tt.want, th.Border)
		})
	}
}
