// Package notification provides transient, non-interactive TUI notifications.
package notification

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
)

// ShowMsg requests a notification. Use Success, Info, or Error to create one.
type ShowMsg struct {
	Text    string
	IsError bool
}

type expiredMsg struct{ id uint64 }

// Success returns a command that displays a success notification.
func Success(text string) tea.Cmd { return Info(text) }

// Info returns a command that displays an informational notification.
func Info(text string) tea.Cmd {
	return func() tea.Msg { return ShowMsg{Text: text} }
}

// Error returns a command that displays an error notification.
// A nil error produces no command.
func Error(err error) tea.Cmd {
	if err == nil {
		return nil
	}
	return func() tea.Msg { return ShowMsg{Text: err.Error(), IsError: true} }
}

// Model owns a single notification. New messages replace the current message.
type Model struct {
	message ShowMsg
	id      uint64
	styles  Styles
}

// Styles defines notification colors independently of the host application.
// Nil colors leave the terminal's corresponding default color unchanged.
type Styles struct {
	Foreground  color.Color
	Background  color.Color
	InfoBorder  color.Color
	ErrorBorder color.Color
}

// DefaultStyles returns notification colors suitable for an unthemed terminal.
func DefaultStyles() Styles {
	return Styles{
		InfoBorder:  lipgloss.Cyan,
		ErrorBorder: lipgloss.Red,
	}
}

// New creates a notification model with the supplied styles.
func New(styles Styles) *Model { return &Model{styles: styles} }

// Update handles notification requests and expiry messages.
// The boolean reports whether the message belongs to this component.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case ShowMsg:
		m.id++
		msg.Text = sanitize.TerminalText(msg.Text)
		m.message = msg
		if msg.Text == "" {
			return nil, true
		}
		lifetime := 3 * time.Second
		if msg.IsError {
			lifetime = 6 * time.Second
		}
		id := m.id
		return tea.Tick(lifetime, func(time.Time) tea.Msg { return expiredMsg{id: id} }), true
	case expiredMsg:
		if msg.id == m.id {
			m.message = ShowMsg{}
		}
		return nil, true
	default:
		return nil, false
	}
}

// Overlay draws the current notification above content without changing its size.
// On terminals too small for a bordered notification, only content is shown.
func (m *Model) Overlay(content string) string {
	width, height := lipgloss.Width(content), lipgloss.Height(content)
	if m.message.Text == "" || width < 8 || height < 3 {
		return content
	}
	color := m.styles.InfoBorder
	label := "✓ "
	if m.message.IsError {
		color, label = m.styles.ErrorBorder, "! "
	}
	style := lipgloss.NewStyle().
		Foreground(m.styles.Foreground).Background(m.styles.Background).
		Border(lipgloss.RoundedBorder()).BorderForeground(color).Padding(0, 1)
	text := (truncate.Right{}).MaxWidth(min(50, width-4)).Render(label + m.message.Text)
	toast := style.Render(text)
	x := max(0, width-lipgloss.Width(toast)-1)
	y := max(0, height-lipgloss.Height(toast)-1)
	rendered := lipgloss.NewCompositor(
		lipgloss.NewLayer(content),
		lipgloss.NewLayer(toast).X(x).Y(y).Z(1),
	).Render()
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, rendered)
}
