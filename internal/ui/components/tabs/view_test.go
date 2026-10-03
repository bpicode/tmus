package tabs_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/tabs"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestView(t *testing.T) {
	tests := []struct {
		name  string
		items []tabs.Tab
		width int
		want  string
	}{
		{
			name: "joined borders", items: []tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}}, width: 13,
			want: "┌─────┬─────┐\n│ One │ Two │\n├─────┴─────┤",
		},
		{
			name: "remaining width", items: []tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}}, width: 15,
			want: "┌─────┬───────┐\n│ One │ Two   │\n├─────┴───────┤",
		},
		{
			name: "no tabs", width: 8,
			want: "┌──────┐\n│      │\n├──────┤",
		},
		{
			name: "empty labels", items: []tabs.Tab{{ID: "one"}, {ID: "two"}}, width: 7,
			want: "┌──┬──┐\n│  │  │\n├──┴──┤",
		},
		{
			name: "unicode", items: []tabs.Tab{{ID: "one", Label: "音楽"}, {ID: "two", Label: "é"}, {ID: "three", Label: "👩‍💻"}}, width: 17,
			want: "┌──────┬───┬────┐\n│ 音楽 │ é │ 👩‍💻 │\n├──────┴───┴────┤",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New(tt.items)
			require.NoError(t, err)
			m.SetSize(tt.width, 3)
			assert.Equal(t, tt.want, ansi.Strip(m.View()))
		})
	}
}

func TestViewPreservesLabelColors(t *testing.T) {
	label := lipgloss.NewStyle().Foreground(lipgloss.Red).Render("音楽 🎵")
	m, err := tabs.New([]tabs.Tab{{ID: "one", Label: label}, {ID: "two", Label: "Go"}})
	require.NoError(t, err)
	m.SetSize(16, 3)
	view := m.View()
	assert.Equal(t, "┌─────────┬────┐\n│ 音楽 🎵 │ Go │\n├─────────┴────┤", ansi.Strip(view))
	assert.Contains(t, view, label)
	require.NoError(t, m.Select("two"))
	assert.Contains(t, m.View(), label, "inactive labels retain embedded colors too")
}

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		content string
		rows    []string
	}{
		{name: "empty", rows: []string{"│           │", "│           │"}},
		{name: "padded", content: "hi", rows: []string{"│hi         │", "│           │"}},
		{name: "multiple lines", content: "hi\nthere", rows: []string{"│hi         │", "│there      │"}},
		{name: "clipped", content: "123456789012345\nnext\nignored", rows: []string{"│12345678901│", "│next       │"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}})
			require.NoError(t, err)
			m.SetSize(13, 6)
			width, height := m.ContentSize()
			assert.Equal(t, 11, width)
			assert.Equal(t, 2, height)
			want := "┌─────┬─────┐\n│ One │ Two │\n├─────┴─────┤\n" + strings.Join(tt.rows, "\n") + "\n└───────────┘"
			assert.Equal(t, want, ansi.Strip(m.Render(tt.content)))
		})
	}
}

func TestRenderUnicodeAndColors(t *testing.T) {
	m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "A"}})
	require.NoError(t, err)
	m.SetSize(6, 7)
	color := lipgloss.NewStyle().Foreground(lipgloss.Red)
	view := m.Render("音楽界\né👩‍💻xy\n" + color.Render("abcdef"))
	assert.Equal(t, "┌────┐\n│ A  │\n├────┤\n│音楽│\n│é👩‍💻x│\n│abcd│\n└────┘", ansi.Strip(view))
	assert.Contains(t, view, color.Render("abcd"))
}

func TestStylesAndSelection(t *testing.T) {
	m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}})
	require.NoError(t, err)
	m.SetSize(13, 5)
	m.Styles.ActiveTab = lipgloss.NewStyle().Foreground(lipgloss.Red).Bold(true)
	m.Styles.InactiveTab = lipgloss.NewStyle().Foreground(lipgloss.Blue).Italic(true)
	m.Styles.Border = m.Styles.Border.BorderForeground(lipgloss.Green)
	view := m.View()
	assert.Contains(t, view, m.Styles.ActiveTab.Render("One"))
	assert.Contains(t, view, m.Styles.InactiveTab.Render("Two"))
	assert.Contains(t, view, "\x1b[32m┌", "the header border must be green")
	assert.Contains(t, m.Render("body"), lipgloss.NewStyle().Foreground(lipgloss.Green).Render("└───────────┘"))
	require.NoError(t, m.Select("two"))
	view = m.View()
	assert.Contains(t, view, m.Styles.InactiveTab.Render("One"))
	assert.Contains(t, view, m.Styles.ActiveTab.Render("Two"))
}

func TestBorderShapes(t *testing.T) {
	tests := []struct {
		name   string
		border lipgloss.Border
		want   string
	}{
		{
			name: "normal", border: lipgloss.NormalBorder(),
			want: "┌─────┬─────┐\n│ One │ Two │\n├─────┴─────┤\n│body       │\n└───────────┘",
		},
		{
			name: "rounded", border: lipgloss.RoundedBorder(),
			want: "╭─────┬─────╮\n│ One │ Two │\n├─────┴─────┤\n│body       │\n╰───────────╯",
		},
		{
			name: "double", border: lipgloss.DoubleBorder(),
			want: "╔═════╦═════╗\n║ One ║ Two ║\n╠═════╩═════╣\n║body       ║\n╚═══════════╝",
		},
		{
			name: "ASCII", border: lipgloss.ASCIIBorder(),
			want: "+-----+-----+\n| One | Two |\n+-----+-----+\n|body       |\n+-----------+",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}})
			require.NoError(t, err)
			m.SetSize(13, 5)
			configured := lipgloss.NewStyle().Border(tt.border)
			m.Styles.Border = configured
			assert.Equal(t, tt.want, ansi.Strip(m.Render("body")))
			assert.Equal(t, configured, m.Styles.Border)
		})
	}
}

func TestNativeBorderColors(t *testing.T) {
	m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}})
	require.NoError(t, err)
	configured := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderTopForeground(lipgloss.Red).
		BorderRightForeground(lipgloss.Blue).
		BorderBottomBackground(lipgloss.Yellow).
		BorderLeftForeground(lipgloss.Green)
	m.Styles.Border = configured
	m.SetSize(13, 5)
	view := m.Render("body")
	assert.Contains(t, view, "\x1b[31m╭")
	assert.Contains(t, view, lipgloss.NewStyle().Foreground(lipgloss.Blue).Render("│"))
	assert.Contains(t, view, lipgloss.NewStyle().Foreground(lipgloss.Green).Render("│"))
	assert.Contains(t, view, lipgloss.NewStyle().Background(lipgloss.Yellow).Render("╰───────────╯"))
	assert.Equal(t, configured, m.Styles.Border)
}

func TestRenderingResize(t *testing.T) {
	m, err := tabs.New([]tabs.Tab{{ID: "one", Label: "One"}, {ID: "two", Label: "Two"}, {ID: "three", Label: "Three"}})
	require.NoError(t, err)
	require.NoError(t, m.Select("two"))
	m.Focus()
	m.SetSize(21, 6)
	view := m.Render("content")
	m.SetSize(25, 8)
	width, height := m.ContentSize()
	assert.Equal(t, 23, width)
	assert.Equal(t, 4, height)
	assert.Equal(t, 25, lipgloss.Width(m.Render("content")))
	assert.Equal(t, 8, lipgloss.Height(m.Render("content")))
	m.SetSize(21, 6)
	assert.Equal(t, view, m.Render("content"))
	assert.Equal(t, "two", m.ActiveID())
	assert.True(t, m.Focused())
}
