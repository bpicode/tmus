package tabs

import "charm.land/lipgloss/v2"

// Styles defines tab text styles and the shared frame style. Clients supply
// text appearance and border settings; the component supplies padding, sizing,
// and visible border edges. Styles should not add conflicting layout rules.
type Styles struct {
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	// Border configures the frame shape and native border colors. Its shape
	// should provide single-cell edges, corners, and junctions.
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
