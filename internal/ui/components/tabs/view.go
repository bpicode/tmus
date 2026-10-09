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
// below the tabs. Labels retain their natural widths; extra space follows the
// last tab's separator. When the tabs do not fit, the header extends beyond the
// assigned width for the terminal to clip.
// It returns an empty string below three columns or three rows.
func (m Model) View() string {
	if m.width < 3 || m.height < headerHeight {
		return ""
	}
	if len(m.tabs) == 0 {
		return m.Styles.headerStyle(true, true, true).Width(m.width).Render("")
	}
	renderedTabs := make([]string, 0, len(m.tabs)+1)
	used := 0
	for i, tab := range m.tabs {
		style := m.Styles.InactiveTab
		if i == m.active {
			style = m.Styles.ActiveTab
		}
		text := style.Render(tab.Label)
		last := i == len(m.tabs)-1
		rendered := m.Styles.headerStyle(i == 0, last, false).Render(text)
		remaining := m.width - used - lipgloss.Width(rendered)
		if last && remaining == 0 {
			rendered = m.Styles.headerStyle(i == 0, true, true).Render(text)
		}
		renderedTabs = append(renderedTabs, rendered)
		if last && remaining > 0 {
			renderedTabs = append(renderedTabs, m.Styles.fillerStyle().Width(remaining).Render(""))
		}
		used += lipgloss.Width(rendered)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// Render encloses already-rendered content below the tab bar in a shared frame.
// It pads or clips content to ContentSize without wrapping or scrolling.
// When the tabs fit, the result occupies the assigned size. It returns an empty
// string below three columns or five rows. ANSI styles and Unicode cell widths
// are preserved. When the tabs overflow, the content's right border and
// bottom-right corner are omitted. ContentSize is unchanged.
func (m Model) Render(content string) string {
	width, height := m.ContentSize()
	if width == 0 {
		return ""
	}
	// Clip before applying the frame's width, which would otherwise wrap text.
	content = lipgloss.NewStyle().Height(height).MaxHeight(height).MaxWidth(width).Render(content)
	header := m.View()
	fits := lipgloss.Width(header) <= m.width
	body := m.Styles.Border.Border(m.Styles.Border.GetBorderStyle(), false, fits, true, true).Width(m.width).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}
