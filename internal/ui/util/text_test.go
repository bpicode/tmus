package util

import (
	"fmt"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestTruncateLeft(t *testing.T) {
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
		{input: styling.Render("colored text"), maxWidth: 10, expected: styling.Render("…ored text")},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("input=%s maxWidth=%d => expected=%s", tt.input, tt.maxWidth, tt.expected), func(t *testing.T) {
			actual := TruncateLeft(tt.input, tt.maxWidth)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
