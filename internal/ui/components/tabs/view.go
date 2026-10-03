package tabs

import "charm.land/lipgloss/v2"

// The header contains a top border, one label row, and a separator.
const headerHeight = 3

// SetSize sets the total width and height in terminal cells, including the
// frame. Negative dimensions are treated as zero. The parent controls sizing;
// Update does not automatically apply terminal window size messages.
func (m *Model) SetSize(width, height int) {
	m.width = max(0, width)
	m.height = max(0, height)
}

// ContentSize returns the space inside Render's frame, after reserving two
// columns for the sides, three rows for the header, and one for the bottom.
// It returns (0, 0) when the frame and one content row cannot fit.
func (m Model) ContentSize() (width, height int) {
	if m.width < 3 || m.height < headerHeight+2 {
		return 0, 0
	}
	return m.width - 2, m.height - headerHeight - 1
}

// View renders the three-row tab bar, including the top border and the rule
// below the tabs. Labels are rendered as supplied. The assigned width must fit
// all tabs; extra space extends the last tab. There is no overflow handling.
// It returns an empty string below three columns or three rows.
func (m Model) View() string {
	if m.width < 3 || m.height < headerHeight {
		return ""
	}
	if len(m.tabs) == 0 {
		return m.Styles.headerStyle(true, true).Width(m.width).Render("")
	}
	renderedTabs := make([]string, len(m.tabs))
	used := 0
	for i, tab := range m.tabs {
		style := m.Styles.InactiveTab
		if i == m.active {
			style = m.Styles.ActiveTab
		}
		text := style.Render(tab.Label)
		cell := m.Styles.headerStyle(i == 0, i == len(m.tabs)-1)
		if i == len(m.tabs)-1 {
			cell = cell.Width(m.width - used)
		}
		renderedTabs[i] = cell.Render(text)
		used += lipgloss.Width(renderedTabs[i])
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// Render encloses already-rendered content below the tab bar in a shared frame.
// It pads or clips content to ContentSize without wrapping or scrolling.
// When the tabs fit, the result occupies the assigned size. It returns an empty
// string below three columns or five rows. ANSI styles and Unicode cell widths
// are preserved.
func (m Model) Render(content string) string {
	width, height := m.ContentSize()
	if width == 0 {
		return ""
	}
	// Clip before applying the frame's width, which would otherwise wrap text.
	content = lipgloss.NewStyle().Height(height).MaxHeight(height).MaxWidth(width).Render(content)
	body := m.Styles.Border.Border(m.Styles.Border.GetBorderStyle(), false, true, true, true).Width(m.width).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, m.View(), body)
}
