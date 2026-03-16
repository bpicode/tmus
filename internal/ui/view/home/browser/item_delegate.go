package browser

import (
	"fmt"
	"io"
	"path/filepath"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type itemDelegate struct{}

func newEntryDelegate() list.ItemDelegate {
	return itemDelegate{}
}

func (itemDelegate) Height() int {
	return 1
}

func (itemDelegate) Spacing() int {
	return 0
}

func (itemDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	entry, ok := item.(browserListItem)
	if !ok {
		return
	}

	name := entry.entry.Name
	if entry.entry.IsDir {
		name = "📁 " + entry.entry.Name + string(filepath.Separator)
	}
	if entry.entry.IsArchive {
		name = "📦 " + entry.entry.Name
	}
	if entry.entry.IsAudio {
		name = "🎵 " + entry.entry.Name
	}
	name = ansi.Truncate(name, max(0, m.Width()), "…")

	line := name
	if index == m.Index() {
		line = styleSelected.Render(name)
	} else if entry.entry.IsDir {
		line = styleDir.Render(name)
	} else if entry.entry.IsArchive {
		line = styleArchive.Render(name)
	}
	_, _ = fmt.Fprint(w, line)
}
