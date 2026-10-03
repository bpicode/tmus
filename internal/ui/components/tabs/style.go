package tabs

import "charm.land/lipgloss/v2"

// Styles defines tab text styles and the shared frame style. Tab styles configure
// text attributes such as color, bold, and italic. They must not set dimensions,
// padding, margins, borders, or introduce line breaks. The component controls
// layout and visible border edges.
//
// With Lip Gloss v2.0.6, underline and strikethrough can corrupt ANSI escape
// sequences in precolored labels. Use other text attributes for those labels.
type Styles struct {
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	// Border configures the frame shape and native border colors, without text
	// or layout attributes. Its shape must provide single-cell edges, corners,
	// and junctions.
	Border lipgloss.Style
}

// DefaultStyles returns a normal border and a bold active tab.
func DefaultStyles() Styles {
	return Styles{
		ActiveTab: lipgloss.NewStyle().Bold(true),
		Border:    lipgloss.NewStyle().Border(lipgloss.NormalBorder()),
	}
}

func (s Styles) headerStyle(first, last bool) lipgloss.Style {
	border := s.Border.GetBorderStyle()
	border.BottomLeft = border.MiddleLeft
	if last {
		border.BottomRight = border.MiddleRight
	} else {
		border.TopRight = border.MiddleTop
		border.BottomRight = border.MiddleBottom
		border.Right = border.Left
	}
	return s.Border.Border(border, true, true, true, first).Padding(0, 1)
}
