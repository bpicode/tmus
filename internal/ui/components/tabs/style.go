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

func (s Styles) headerStyle(first, last, frameEdge bool) lipgloss.Style {
	border := s.Border.GetBorderStyle()
	border.BottomLeft = border.MiddleLeft
	border.BottomRight = border.MiddleBottom
	if frameEdge {
		border.BottomRight = border.MiddleRight
	}
	if !last {
		border.TopRight = border.MiddleTop
		border.Right = border.Left
	}
	return s.Border.Border(border, true, true, true, first).Padding(0, 1)
}

func (s Styles) fillerStyle() lipgloss.Style {
	border := s.Border.GetBorderStyle()
	// Keep a right border so Lip Gloss renders and styles the corner, but leave
	// the two rows above the separator blank.
	border.Right = " "
	border.BottomRight = border.TopRight
	return s.Border.Border(border, false, true, true, false).Padding(1, 0, 0, 0)
}
