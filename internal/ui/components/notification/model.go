// Package notification provides transient, non-interactive TUI notifications.
package notification

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
)

type msgType int

const (
	msgTypeInfo msgType = iota
	msgTypeSuccess
	msgTypeWarn
	msgTypeError
)

// Msg requests a notification. Use Success, Info, or Error to create one.
type Msg struct {
	Text    string
	msgType msgType
}

type expiredMsg struct{ id uint64 }

// Info returns a command that displays an informational notification.
func Info(text string) tea.Cmd {
	return func() tea.Msg { return Msg{Text: text, msgType: msgTypeInfo} }
}

// Success returns a command that displays a success notification.
func Success(text string) tea.Cmd {
	return func() tea.Msg { return Msg{Text: text, msgType: msgTypeSuccess} }
}

// Warn returns a command that displays a warning notification.
func Warn(text string) tea.Cmd {
	return func() tea.Msg { return Msg{Text: text, msgType: msgTypeWarn} }
}

// Error returns a command that displays an error notification.
// A nil error produces no command.
func Error(err error) tea.Cmd {
	if err == nil {
		return nil
	}
	return func() tea.Msg { return Msg{Text: err.Error(), msgType: msgTypeError} }
}

// Model owns a single notification. New messages replace the current message.
type Model struct {
	message Msg
	id      uint64
	styles  Styles
}

// Styles defines notification colors independently of the host application.
// Nil colors leave the terminal's corresponding default color unchanged.
type Styles struct {
	Info    lipgloss.Style
	Success lipgloss.Style
	Warn    lipgloss.Style
	Error   lipgloss.Style
}

// DefaultStyles returns notification colors suitable for an unthemed terminal.
func DefaultStyles() Styles {
	return Styles{
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Cyan).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Cyan).Padding(0, 1),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Green).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Green).Padding(0, 1),
		Warn:    lipgloss.NewStyle().Foreground(lipgloss.Yellow).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Yellow).Padding(0, 1),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Red).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Red).Padding(0, 1),
	}
}

type Option func(*Model)

// WithStyles replaces the complete notification styles.
func WithStyles(styles Styles) Option {
	return func(m *Model) {
		m.styles = styles
	}
}

// New creates a notification model with the supplied styles.
func New(opts ...Option) *Model {
	m := &Model{styles: DefaultStyles()}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Update handles notification requests and expiry messages.
// The boolean reports whether the message belongs to this component.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case Msg:
		m.id++
		msg.Text = sanitize.TerminalText(msg.Text)
		m.message = msg
		if msg.Text == "" {
			return nil, true
		}
		lifetime := 2 * time.Second
		if msg.msgType == msgTypeError {
			lifetime = 4 * time.Second
		}
		id := m.id
		return tea.Tick(lifetime, func(time.Time) tea.Msg { return expiredMsg{id: id} }), true
	case expiredMsg:
		if msg.id == m.id {
			m.message = Msg{}
		}
		return nil, true
	default:
		return nil, false
	}
}

// Overlay draws the current notification above content without changing its size.
// On terminals too small for a bordered notification, only content is shown.
func (m *Model) Overlay(content string) string {
	if m.message.Text == "" {
		return content
	}
	width, height := lipgloss.Width(content), lipgloss.Height(content)
	if width < 8 || height < 3 {
		return content
	}
	var style lipgloss.Style
	var icon = ""
	switch m.message.msgType {
	case msgTypeError:
		style = m.styles.Error
		icon = "❌"
	case msgTypeWarn:
		style = m.styles.Warn
		icon = "🔔"
	case msgTypeSuccess:
		style = m.styles.Success
		icon = "✅"
	default:
		style = m.styles.Info
	}
	textWidth := min(50, width-style.GetHorizontalFrameSize())
	if textWidth <= 0 {
		return content
	}
	var text string
	var tr = (truncate.Right{}).MaxWidth(textWidth)
	if icon != "" {
		text = tr.Render(icon, m.message.Text)
	} else {
		text = tr.Render(m.message.Text)
	}
	toast := style.Render(text)
	toastWidth, toastHeight := lipgloss.Width(toast), lipgloss.Height(toast)
	if toastWidth > width || toastHeight > height {
		return content
	}
	x := max(0, width-toastWidth-2)
	y := max(0, height-toastHeight-1)
	rendered := lipgloss.NewCompositor(
		lipgloss.NewLayer(content),
		lipgloss.NewLayer(toast).X(x).Y(y).Z(1),
	).Render()
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, rendered)
}
