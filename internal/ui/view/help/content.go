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

func (h *content) render() []string {
	maxWidthKey := 0
	for _, s := range h.sections {
		for _, hk := range s.helpKeys {
			maxWidthKey = max(maxWidthKey, hk.width())
		}
	}

	keyPadLeft := 4
	keyFillMiddle := maxWidthKey + 4

	lines := []string{
		styleTitle.Render(h.title),
		"",
	}
	for _, s := range h.sections {
		lines = append(lines, s.render(keyPadLeft, keyFillMiddle)...)
		lines = append(lines, "")
	}
	lines = append(lines, h.appendix)
	return lines
}

type helpSection struct {
	subtitle string
	helpKeys []helpKey
}

func (h *helpSection) render(keyPadLeft, keyFillMiddle int) []string {
	lines := []string{
		styleSubtitle.Render(h.subtitle),
	}
	for _, k := range h.helpKeys {
		lines = append(lines, k.render(keyPadLeft, keyFillMiddle))
	}
	return lines
}

type helpKey struct {
	key1     string
	key2     string
	helpText string
}

func (h *helpKey) render(padLeft, fillMiddle int) string {
	paddingLeft := strings.Repeat(" ", max(padLeft, 0))
	paddingMiddle := strings.Repeat(" ", max(fillMiddle-h.width(), 0))
	keys := h.renderKeys()
	return paddingLeft + keys + paddingMiddle + h.helpText
}

func (h *helpKey) renderKeys() string {
	if h.key2 == "" {
		return styleHelpKey.Render(h.key1)
	}
	return styleHelpKey.Render(h.key1) + " / " + styleHelpKey.Render(h.key2)
}

func (h *helpKey) width() int {
	rendered := h.renderKeys()
	return lipgloss.Width(rendered)
}

var keybindings = content{
	title: "📖 tmus keybindings",
	sections: []helpSection{
		{
			subtitle: "🧭 Navigation",
			helpKeys: []helpKey{
				{
					key1:     "tab",
					helpText: "switch focus",
				},
				{
					key1:     "↑/↓",
					key2:     "k/j",
					helpText: "move selection",
				},
				{
					key1:     "pgup",
					key2:     "pgdn",
					helpText: "page selection",
				},
				{
					key1:     "home",
					key2:     "end",
					helpText: "jump to top/bottom",
				},
				{
					key1:     "enter",
					helpText: "open dir / add item",
				},
				{
					key1:     "/",
					helpText: "search in browser",
				},
			},
		},
		{
			subtitle: "🎵 Playback",
			helpKeys: []helpKey{
				{
					key1:     "enter",
					helpText: "play",
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
			},
		},
		{
			subtitle: "📋 Playlist",
			helpKeys: []helpKey{
				{
					key1:     "a",
					helpText: "add file",
				},
				{
					key1:     "A",
					helpText: "add all files",
				},
				{
					key1:     "i",
					helpText: "track info",
				},
				{
					key1:     "/",
					helpText: "search playlist",
				},
				{
					key1:     "x",
					helpText: "remove item",
				},
				{
					key1:     "c",
					helpText: "clear playlist",
				},
			},
		},
		{
			subtitle: "📜 Lyrics",
			helpKeys: []helpKey{
				{
					key1:     "L",
					helpText: "show/hide lyrics",
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
					key1:     "b",
					helpText: "toggle browser",
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
			subtitle: "☕ Other",
			helpKeys: []helpKey{
				{
					key1:     "?",
					helpText: "toggle this help",
				},
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
