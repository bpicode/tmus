package truncate

import (
	"fmt"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestLeft(t *testing.T) {
	var styling = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	var tests = []struct {
		input    string
		maxWidth int
		expected string
	}{
		{input: "foobar", maxWidth: 10, expected: "foobar"},
		{input: "foobar", maxWidth: 6, expected: "foobar"},
		{input: "foobar", maxWidth: 5, expected: "…obar"},
		{input: "foobar", maxWidth: 1, expected: "…"},
		{input: "foobar", maxWidth: 0, expected: ""},
		{input: "foobar", maxWidth: -1, expected: ""},
		{input: "foobar", maxWidth: -42, expected: ""},
		{input: "foobar\nfoo\nbar\nbarbaz", maxWidth: 3, expected: "…ar\nfoo\nbar\n…az"},
		{input: styling.Render("colored text"), maxWidth: 10, expected: styling.Render("…ored text")},
		{input: styling.Render("foobar\nbarbaz"), maxWidth: 4, expected: styling.Render("…bar\n…baz")},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("input=%s maxWidth=%d => expected=%s", tt.input, tt.maxWidth, tt.expected), func(t *testing.T) {
			left := Left{Style: lipgloss.NewStyle()}.MaxWidth(tt.maxWidth)
			actual := left.Render(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
