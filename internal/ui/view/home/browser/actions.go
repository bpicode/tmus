package browser

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/library"
)

type loadDirMsg struct {
	Path  string
	Items []library.Entry
	Err   error
}

func loadDirCmd(path string, showHidden bool) tea.Cmd {
	return func() tea.Msg {
		items, err := library.List(path, showHidden)
		return loadDirMsg{
			Path:  path,
			Items: items,
			Err:   err,
		}
	}
}
