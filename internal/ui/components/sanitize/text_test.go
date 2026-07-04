package sanitize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerminalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Master of Carpets",
			want:  "Master of Carpets",
		},
		{
			name:  "strips ANSI SGR",
			input: "\x1b[31mred\x1b[0m",
			want:  "red",
		},
		{
			name:  "strips OSC hyperlink",
			input: "\x1b]8;;https://example.invalid\x1b\\link\x1b]8;;\x1b\\",
			want:  "link",
		},
		{
			name:  "normalizes whitespace",
			input: "one\ttwo\r\nthree\nfour\rfive",
			want:  "one    two three four five",
		},
		{
			name:  "removes C0 and C1 controls",
			input: "a\x00b\x1fc\u0085d\x7fe",
			want:  "abcde",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TerminalText(tt.input))
		})
	}
}
