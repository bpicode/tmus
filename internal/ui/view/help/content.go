package help

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type content struct {
	title    string
	sections []helpSection
	appendix string
}

func (h *content) render(styles styles) []string {
	maxWidthKey := 0
	for _, s := range h.sections {
		for _, hk := range s.helpKeys {
			maxWidthKey = max(maxWidthKey, hk.width(styles))
		}
	}

	keyPadLeft := 4
	keyFillMiddle := maxWidthKey + 4

	lines := []string{
		styles.title.Render(h.title),
		"",
	}
	for _, s := range h.sections {
		lines = append(lines, s.render(keyPadLeft, keyFillMiddle, styles)...)
		lines = append(lines, "")
	}
	return lines
}

type helpSection struct {
	subtitle string
	helpKeys []helpKey
}

func (h *helpSection) render(keyPadLeft, keyFillMiddle int, styles styles) []string {
	lines := []string{
		styles.subtitle.Render(h.subtitle),
	}
	for _, k := range h.helpKeys {
		lines = append(lines, k.render(keyPadLeft, keyFillMiddle, styles))
	}
	return lines
}

type helpKey struct {
	key1     string
	key2     string
	helpText string
}

func (h *helpKey) render(padLeft, fillMiddle int, styles styles) string {
	paddingLeft := strings.Repeat(" ", max(padLeft, 0))
	paddingMiddle := strings.Repeat(" ", max(fillMiddle-h.width(styles), 0))
	keys := h.renderKeys(styles)
	return paddingLeft + keys + paddingMiddle + h.helpText
}

func (h *helpKey) renderKeys(styles styles) string {
	if h.key2 == "" {
		return styles.helpKey.Render(h.key1)
	}
	return styles.helpKey.Render(h.key1) + " / " + styles.helpKey.Render(h.key2)
}

func (h *helpKey) width(styles styles) int {
	rendered := h.renderKeys(styles)
	return lipgloss.Width(rendered)
}

var keybindings = content{
	title: "📖 Keybindings",
	sections: []helpSection{
		{
			subtitle: "📑 Tabs",
			helpKeys: []helpKey{
				{
					key1:     "tab",
					key2:     "shift+tab",
					helpText: "next / previous tab",
				},
				{
					key1:     "b",
					helpText: "switch between Playlist and Browser",
				},
				{
					key1:     "?",
					helpText: "open Help / return to Playlist",
				},
				{
					key1:     "esc",
					helpText: "return to Playlist from other tabs",
				},
			},
		},
		{
			subtitle: "🧭 Navigation (active tab)",
			helpKeys: []helpKey{
				{
					key1:     "↑/↓",
					key2:     "k/j",
					helpText: "move selection / scroll",
				},
				{
					key1:     "pgup",
					key2:     "pgdn",
					helpText: "page selection / scroll",
				},
				{
					key1:     "home",
					key2:     "end",
					helpText: "jump to top/bottom",
				},
			},
		},
		{
			subtitle: "🎧 Playlist",
			helpKeys: []helpKey{
				{
					key1:     "enter",
					helpText: "play selected track",
				},
				{
					key1:     "space",
					helpText: "pause / resume",
				},
				{
					key1:     "n",
					key2:     "p",
					helpText: "next / prev",
				},
				{
					key1:     "s",
					helpText: "stop",
				},
				{
					key1:     "M",
					helpText: "cycle play mode",
				},
				{
					key1:     "+",
					key2:     "-",
					helpText: "volume up / down",
				},
				{
					key1:     "m",
					helpText: "mute / unmute",
				},
				{
					key1:     ",",
					key2:     ".",
					helpText: "seek -10s / +10s",
				},
				{
					key1:     "<",
					key2:     ">",
					helpText: "seek -60s / +60s",
				},
				{
					key1:     "i",
					helpText: "show track information (opens 'Track' tab)",
				},
				{
					key1:     "L",
					helpText: "show lyrics of selected track (opens 'Lyrics' tab)",
				},
				{
					key1:     "x",
					key2:     "delete",
					helpText: "remove item",
				},
				{
					key1:     "c",
					helpText: "clear playlist",
				},
				{
					key1:     "alt+↑",
					key2:     "alt+k",
					helpText: "move item up",
				},
				{
					key1:     "alt+↓",
					key2:     "alt+j",
					helpText: "move item down",
				},
			},
		},
		{
			subtitle: "🎵 Track",
			helpKeys: []helpKey{
				{
					key1:     "i",
					key2:     "esc",
					helpText: "return to Playlist",
				},
			},
		},
		{
			subtitle: "📜 Lyrics",
			helpKeys: []helpKey{
				{
					key1:     "L",
					key2:     "esc",
					helpText: "return to Playlist",
				},
				{
					key1:     "f",
					helpText: "follow/unfollow lyrics",
				},
			},
		},
		{
			subtitle: "📂 Browser",
			helpKeys: []helpKey{
				{
					key1:     "enter",
					helpText: "open directory / archive, or add audio file",
				},
				{
					key1:     "backspace",
					helpText: "go to parent directory",
				},
				{
					key1:     "a",
					helpText: "add selected audio file to playlist",
				},
				{
					key1:     "A",
					helpText: "add all visible audio files to playlist",
				},
				{
					key1:     "ctrl+r",
					helpText: "reload current directory",
				},
				{
					key1:     "H",
					helpText: "toggle hidden files",
				},
				{
					key1:     "~",
					helpText: "go to home directory",
				},
			},
		},
		{
			subtitle: "🔎 Search (where applicable)",
			helpKeys: []helpKey{
				{
					key1:     "/",
					helpText: "start search",
				},
				{
					key1:     "enter",
					helpText: "apply search",
				},
				{
					key1:     "esc",
					helpText: "cancel / clear search",
				},
			},
		},
		{
			subtitle: "🚪 Quit",
			helpKeys: []helpKey{
				{
					key1:     "q",
					key2:     "ctrl+c",
					helpText: "quit",
				},
			},
		},
	},
	appendix: "(j/k or ↑/↓ to scroll, esc to close)",
}
