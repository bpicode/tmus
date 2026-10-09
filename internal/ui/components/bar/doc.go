// Package bar provides horizontal divider bars for terminal UIs.
//
// A bar renders as a full-width horizontal rule of "─" characters, optionally
// carrying a piece of text embedded in the rule. Bars are constructed with
// [Horizontal], configured through functional options:
//
//	bar.Horizontal(width, bar.WithStyle(style), bar.WithText("Title", bar.PlacementCenter)).View()
//
// # Text placement
//
// [WithText] positions the text within the bar via a [TextPlacement]:
//
//   - [PlacementLeft]: text sits near the left edge, surrounded by dashes.
//     Padding is balanced between both sides, but at most [WithMaxEdgePad]
//     (2 by default) dashes appear on the left of the text.
//   - [PlacementCenter]: text is centered; an odd leftover dash goes right.
//   - [PlacementRight]: text sits near the right edge. Padding is balanced
//     between both sides, but at most [WithMaxEdgePad] (2 by default) dashes
//     appear on the right of the text. Unknown placement values degrade to
//     centered.
//
// # Overflow
//
// Text wider than the bar is truncated with a leading or trailing ellipsis:
// placements other than [PlacementRight] keep the start of the text
// ("Hello, w…"), while [PlacementRight] keeps the end ("…, world") since the
// text is anchored at the right edge.
//
// Bars carry no Bubble Tea state; [HBar.View] renders deterministically from
// the configured width, style, text, and placement.
package bar
