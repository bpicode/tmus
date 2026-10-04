package browser

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
)

type itemDelegate struct {
	styles styles
}

func newEntryDelegate(styles styles) list.ItemDelegate {
	return itemDelegate{styles: styles}
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

func (i itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	entry, ok := item.(browserListItem)
	if !ok {
		return
	}

	ext := strings.ToLower(filepath.Ext(entry.entry.Path()))
	isStream := ext == ".url" || ext == ".stream"

	entryName := sanitize.TerminalText(entry.entry.Name())
	name := entryName
	if entry.entry.IsDir() {
		name = "📁 " + entryName + string(filepath.Separator)
	}
	if entry.entry.IsArchive() {
		name = "📦 " + entryName
	}
	if entry.entry.IsAudio() {
		if isStream {
			name = "📻 " + entryName
		} else {
			name = "🎵 " + entryName
		}
	}

	style := lipgloss.NewStyle()
	if index == m.Index() {
		style = i.styles.selected
	} else if entry.entry.IsDir() {
		style = i.styles.dir
	} else if entry.entry.IsArchive() {
		style = i.styles.archive
	}
	truncateRight := truncate.Right{Style: style}.MaxWidth(m.Width())
	_, _ = fmt.Fprint(w, truncateRight.Render(name))
}
