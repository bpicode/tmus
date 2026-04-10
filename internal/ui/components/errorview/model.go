package errorview

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type Model struct {
	err    error
	styles Styles
}

type Styles struct {
	Error lipgloss.Style
}

func New(styles Styles) *Model {
	return &Model{
		styles: styles,
	}
}

func (m *Model) SetErr(err error) {
	m.err = err
}

func (m *Model) HasErr() bool {
	return m.err != nil
}

func (m *Model) View() string {
	if m.err == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(m.viewHeader())
	unwrapped := m.unwrapErr()
	if len(unwrapped) > 0 {
		sb.WriteString("\n")
	}
	for i, errItem := range unwrapped {
		sb.WriteString(m.styles.Error.Render("    → " + errItem.Error()))
		if i < len(unwrapped)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *Model) viewHeader() string {
	return m.styles.Error.Render("❌ " + m.err.Error())
}

func (m *Model) unwrapErr() []error {
	if unwrapper, ok := m.err.(interface{ Unwrap() []error }); ok {
		return unwrapper.Unwrap()
	}
	return nil
}
