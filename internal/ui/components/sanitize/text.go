package sanitize

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// TerminalText removes terminal control sequences and control characters from
// untrusted text before it is rendered by the TUI.
func TerminalText(value string) string {
	if value == "" {
		return ""
	}

	value = ansi.Strip(value)
	value = strings.NewReplacer(
		"\t", "    ",
		"\r\n", " ",
		"\r", " ",
		"\n", " ",
	).Replace(value)

	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, value)
}
