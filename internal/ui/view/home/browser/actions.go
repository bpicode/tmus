package browser

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/library"
)

type loadDirMsg struct {
	Path  string
	Items []library.Entry2
	Err   error
}

func loadDirCmd(path string, showHidden bool) tea.Cmd {
	return func() tea.Msg {
		items, err := library.List2(path)
		if err == nil && !showHidden {
			items = filterHiddenEntries(items)
		}
		return loadDirMsg{
			Path:  path,
			Items: items,
			Err:   err,
		}
	}
}

func filterHiddenEntries(entries []library.Entry2) []library.Entry2 {
	items := make([]library.Entry2, 0, len(entries))
	for _, entry := range entries {
		if !entry.Hidden() {
			items = append(items, entry)
		}
	}
	return items
}
