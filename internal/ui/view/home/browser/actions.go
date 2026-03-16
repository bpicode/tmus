package browser

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/library"
)

type LoadDirMsg struct {
	Path  string
	Items []library.Entry
	Err   error
}

func LoadDirCmd(path string, showHidden bool) tea.Cmd {
	return func() tea.Msg {
		items, err := library.List(path, showHidden)
		return LoadDirMsg{
			Path:  path,
			Items: items,
			Err:   err,
		}
	}
}
