package bar

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
)

// TextPlacement describes where the text sits within the bar.
type TextPlacement int

const (
	// PlacementLeft anchors the text near the left edge of the bar.
	PlacementLeft TextPlacement = iota
	// PlacementCenter centers the text within the bar.
	PlacementCenter
	// PlacementRight anchors the text near the right edge of the bar.
	PlacementRight
)

// HBar is a horizontal divider bar with optional embedded text. Create one
// with [Horizontal]; the zero value renders as an empty string.
type HBar struct {
	width      int
	style      lipgloss.Style
	text       string
	placement  TextPlacement
	maxEdgePad int
}

// Horizontal returns a bar of the given cell width, configured via opts.
// A width of zero or less produces a bar that renders as an empty string.
func Horizontal(width int, opts ...Option) HBar {
	bar := HBar{width: width, maxEdgePad: 2}
	for _, opt := range opts {
		bar = opt(bar)
	}
	return bar
}

// Option configures an [HBar]. Options are applied in order by [Horizontal];
// later options override earlier ones.
type Option func(HBar) HBar

// WithStyle sets the style applied to the dash characters of the bar. The
// text itself is rendered as passed in, so pre-style it with a matching style
// if needed.
func WithStyle(style lipgloss.Style) Option {
	return func(h HBar) HBar {
		h.style = style
		return h
	}
}

// WithText embeds text in the bar at the given placement. An empty text
// yields a plain bar of dashes. Text wider than the bar is truncated with an
// ellipsis; see the package documentation for details.
func WithText(text string, placement TextPlacement) Option {
	return func(h HBar) HBar {
		h.text = text
		h.placement = placement
		return h
	}
}

// WithMaxEdgePad sets the maximum number of dashes between the text and the
// bar edge on the text's side (2 by default). Padding is split evenly until
// twice this value is available; beyond that the text stays pinned at the
// edge and remaining dashes accumulate on the far side. Negative values are
// clamped to 0.
func WithMaxEdgePad(maxEdgePad int) Option {
	return func(h HBar) HBar {
		h.maxEdgePad = max(0, maxEdgePad)
		return h
	}
}

// View renders the bar as a single line of exactly the configured width
// (measured in printable cells). It returns an empty string if the width is
// zero or less.
func (h HBar) View() string {
	if h.width <= 0 {
		return ""
	}
	if h.text == "" {
		dashes := strings.Repeat("─", h.width)
		return h.style.Render(dashes)
	}
	text := h.text
	switch h.placement {
	case PlacementRight:
		text = truncate.Left{}.MaxWidth(h.width).Render(text)
	default:
		text = truncate.Right{}.MaxWidth(h.width).Render(text)
	}

	textWidth := lipgloss.Width(text)
	availableForDashes := max(h.width-textWidth, 0)
	if availableForDashes == 0 {
		return text
	}

	leftPad := 0
	rightPad := 0
	switch h.placement {
	case PlacementLeft:
		// try to balance left and right padding, but never more than maxEdgePad dashes on the left
		// examples for maxEdgePad = 2:
		// - availableForDashes = 5, leftPad = 2, rightPad = 3
		// - availableForDashes = 4, leftPad = 2, rightPad = 2
		// - availableForDashes = 3, leftPad = 1, rightPad = 2
		// - availableForDashes = 2, leftPad = 1, rightPad = 1
		// - availableForDashes = 1, leftPad = 0, rightPad = 1
		// - availableForDashes = 0, leftPad = 0, rightPad = 0
		leftPad = min(h.maxEdgePad, availableForDashes/2)
		rightPad = max(availableForDashes-leftPad, 0)
	case PlacementRight:
		// try to balance left and right padding, but never more than maxEdgePad dashes on the right
		rightPad = min(h.maxEdgePad, availableForDashes/2)
		leftPad = max(availableForDashes-rightPad, 0)
	default:
		leftPad = availableForDashes / 2
		rightPad = max(availableForDashes-leftPad, 0)
	}

	left := h.style.Render(strings.Repeat("─", leftPad))
	right := h.style.Render(strings.Repeat("─", rightPad))
	return left + text + right
}
