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

func loadDirCmd(lib *library.Library, path string, showHidden bool) tea.Cmd {
	if lib == nil {
		lib = library.New(library.DefaultOptions())
	}
	return func() tea.Msg {
		items, err := lib.List(path)
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

func filterHiddenEntries(entries []library.Entry) []library.Entry {
	items := make([]library.Entry, 0, len(entries))
	for _, entry := range entries {
		if !entry.Hidden() {
			items = append(items, entry)
		}
	}
	return items
}
