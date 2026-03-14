package util

import (
	"github.com/charmbracelet/x/ansi"
)

func TruncateLeft(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	width := ansi.StringWidth(s)
	if width <= maxWidth {
		return s
	}
	if maxWidth == 1 {
		return "…"
	}
	toRemove := width - (maxWidth - 1)
	return ansi.TruncateLeft(s, toRemove, "…")
}
