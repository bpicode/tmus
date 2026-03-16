package browser

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/archive"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/ui/util"
)

type Model struct {
	Cwd        string
	entries    []library.Entry
	showHidden bool
	err        error
	width      int
	height     int
	show       bool
	focus      bool
	app        *core.App
	list       list.Model
}

func NewModel(startDir string, appRef *core.App) *Model {
	delegate := newEntryDelegate()
	browserList := list.New(nil, delegate, 0, 0)
	browserList.SetShowTitle(false)
	browserList.SetShowFilter(false)
	browserList.SetShowStatusBar(false)
	browserList.SetShowPagination(true)
	browserList.SetShowHelp(false)
	browserList.SetFilteringEnabled(true)
	browserList.DisableQuitKeybindings()
	browserList.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup", "pageup"), key.WithHelp("pgup", "prev page"))
	browserList.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown", "pagedown"), key.WithHelp("pgdn", "next page"))
	browserList.KeyMap.GoToStart = key.NewBinding(key.WithKeys("home", "pos1"), key.WithHelp("home", "go to start"))
	browserList.KeyMap.GoToEnd = key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "go to end"))
	browserList.FilterInput.Prompt = "Search: "
	browserList.FilterInput.Placeholder = "/"
	browserList.FilterInput.SetStyles(textinput.Styles{
		Focused: textinput.StyleState{Text: styleSearchActive, Prompt: styleSearchActive},
		Blurred: textinput.StyleState{Text: styleSearchInactive, Prompt: styleSearchInactive, Placeholder: styleSearchInactive},
	})
	browserList.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	browserList.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("•")
	browserList.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("·")
	browserList.Paginator.ActiveDot = browserList.Styles.ActivePaginationDot.String()
	browserList.Paginator.InactiveDot = browserList.Styles.InactivePaginationDot.String()

	b := &Model{
		Cwd:  startDir,
		app:  appRef,
		list: browserList,
	}
	return b
}

func (m *Model) loadDir(path string) tea.Cmd {
	m.Cwd = path
	return LoadDirCmd(path, m.showHidden)
}

func (m *Model) loadHomeDir() tea.Cmd {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		m.err = err
		return nil
	}
	return m.loadDir(homeDir)
}

func (m *Model) openSelection() tea.Cmd {
	m.err = nil
	selected, ok := m.selected()
	if !ok || !selected.IsDir {
		if ok && !archive.IsArchivePath(selected.Path) {
			if archivePath, ok := library.OpenArchiveRoot(selected.Path); ok {
				m.clearSearch()
				return m.loadDir(archivePath)
			}
		}
		return nil
	}
	m.clearSearch()
	return m.loadDir(selected.Path)
}

func (m *Model) upDir() tea.Cmd {
	m.err = nil
	if archive.IsArchivePath(m.Cwd) {
		scheme, archivePath, inner, err := archive.SplitPath(m.Cwd)
		if err != nil {
			m.err = err
			return nil
		}
		if inner == "" {
			m.clearSearch()
			return m.loadDir(filepath.Dir(archivePath))
		}
		parent := path.Dir(inner)
		if parent == "." {
			parent = ""
		}
		m.clearSearch()
		return m.loadDir(archive.BuildPath(scheme, archivePath, parent))
	}
	parent := filepath.Dir(m.Cwd)
	if parent == m.Cwd {
		return nil
	}
	m.clearSearch()
	return m.loadDir(parent)
}

func (m *Model) toggleHidden() tea.Cmd {
	m.showHidden = !m.showHidden
	return m.loadDir(m.Cwd)
}

func (m *Model) selected() (library.Entry, bool) {
	item, ok := m.list.SelectedItem().(browserListItem)
	if !ok {
		return library.Entry{}, false
	}
	return item.entry, true
}

func (m *Model) Init() tea.Cmd {
	return LoadDirCmd(m.Cwd, m.showHidden)
}

func (m *Model) View() string {
	var sb strings.Builder
	title := styleTitle
	panelStyle := stylePanelUnfocused.Width(m.width).Height(m.height)
	if m.focus {
		title = styleTitleFocused
		panelStyle = stylePanelFocused.Width(m.width).Height(m.height)
	}
	sb.WriteString(title.Render("📂 Files"))
	sb.WriteString("\n")
	pathWidth := max(0, m.width-panelStyle.GetHorizontalFrameSize())
	cwd := util.TruncateLeft(m.Cwd, pathWidth)
	sb.WriteString(styleCwd.Render(cwd))
	sb.WriteString("\n")
	sb.WriteString(m.searchView())
	sb.WriteString("\n")
	sb.WriteString(styleSeparator.Render(strings.Repeat("─", max(0, m.width-panelStyle.GetHorizontalFrameSize()))))
	sb.WriteString("\n")

	if m.err != nil {
		sb.WriteString(styleError.Render(m.err.Error()))
		return panelStyle.Render(sb.String())
	}

	if len(m.entries) == 0 {
		sb.WriteString(styleEmpty.Render("(empty)"))
		return panelStyle.Render(sb.String())
	}

	headerLines := 6
	availableLines := max(0, m.height-headerLines)
	if availableLines == 0 {
		return panelStyle.Render(sb.String())
	}

	m.list.SetSize(max(0, m.width-panelStyle.GetHorizontalFrameSize()), availableLines)
	sb.WriteString(m.list.View())

	return panelStyle.Render(sb.String())
}

func (m *Model) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.show || !m.focus {
		return nil, false
	}
	if handled, cmd := m.updateSearch(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.updateNav(msg); handled {
		return cmd, true
	}
	return nil, false
}

func (m *Model) updateListItems(preferredIndex int) {
	entries := m.entries
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, browserListItem{entry: entry})
	}
	if cmd := m.list.SetItems(items); cmd != nil {
		m.list, _ = m.list.Update(cmd())
	}
	if len(items) == 0 {
		m.list.Select(0)
		return
	}
	m.list.Select(clamp(preferredIndex, 0, len(items)-1))
}

func (m *Model) updateSearch(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !m.list.SettingFilter() {
		return false, nil
	}
	if msg.String() == "ctrl+c" {
		return false, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncFilterWhileEditing()
	return true, cmd
}

func (m *Model) syncFilterWhileEditing() {
	if !m.list.SettingFilter() {
		return
	}
	// This view only receives key messages from the parent model, so we
	// apply filtering synchronously instead of relying on list filter commands.
	filter := m.list.FilterInput.Value()
	m.list.SetFilterText(filter)
	m.list.SetFilterState(list.Filtering)
}

func (m *Model) clearSearch() {
	m.list.ResetFilter()
}

func (m *Model) searchView() string {
	switch {
	case m.list.SettingFilter():
		return m.list.FilterInput.View()
	case m.list.IsFiltered():
		return styleSearchActive.Render("Search: " + m.list.FilterValue())
	default:
		return styleSearchInactive.Render("Search: /")
	}
}

func (m *Model) visibleEntries() []library.Entry {
	items := m.list.VisibleItems()
	entries := make([]library.Entry, 0, len(items))
	for _, item := range items {
		browserItem, ok := item.(browserListItem)
		if !ok {
			continue
		}
		entries = append(entries, browserItem.entry)
	}
	return entries
}

func (m *Model) updateNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.show || !m.focus {
		return nil, false
	}
	switch msg.String() {
	case "/", "up", "k", "down", "j", "pgup", "pageup", "pgdown", "pagedown", "home", "pos1", "end", "esc":
		m.err = nil
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return cmd, true
	case "enter":
		if selected, ok := m.selected(); ok && !selected.IsDir && selected.IsAudio {
			cmd := core.Command{
				Type:  core.CmdAdd,
				Track: core.Track{Name: selected.Name, Path: selected.Path},
			}
			_ = m.app.Dispatch(cmd)
			return nil, true
		}
		return m.openSelection(), true
	case "backspace", "left", "h":
		return m.upDir(), true
	case "a":
		if selected, ok := m.selected(); ok {
			if !selected.IsDir && selected.IsAudio {
				cmd := core.Command{
					Type:  core.CmdAdd,
					Track: core.Track{Name: selected.Name, Path: selected.Path},
				}
				_ = m.app.Dispatch(cmd)
				return nil, true
			}
		}
		return nil, true
	case "A":
		visible := m.visibleEntries()
		tracks := make([]core.Track, 0, len(visible))
		for _, entry := range visible {
			if entry.IsDir || !entry.IsAudio {
				continue
			}
			tracks = append(tracks, core.Track{Name: entry.Name, Path: entry.Path})
		}
		if len(tracks) > 0 {
			cmd := core.Command{Type: core.CmdAddAll, Tracks: tracks}
			_ = m.app.Dispatch(cmd)
			return nil, true
		}
		return nil, true
	case "ctrl+r":
		return m.loadDir(m.Cwd), true
	case "H":
		return m.toggleHidden(), true
	case "~":
		return m.loadHomeDir(), true
	default:
		return nil, false
	}
}

func (m *Model) UpdateSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.list.SetSize(max(0, m.width-stylePanelFocused.GetHorizontalFrameSize()), max(0, m.height-6))
}

func (m *Model) HandleLoadDirMsg(msg LoadDirMsg) {
	if m.Cwd != msg.Path {
		// This can happen when the user quickly navigates to another directory before the previous one finishes loading.
		// In this case we ignore the message since it's outdated.
		return
	}

	if msg.Err != nil {
		m.err = msg.Err
		return
	}

	prevIndex := m.list.Index()
	m.entries = msg.Items
	m.updateListItems(prevIndex)
	m.err = nil
}

func (m *Model) Visible() bool {
	return m.show
}

func (m *Model) Show(show bool) {
	m.show = show
	if !m.show {
		m.focus = false
	}
}

func (m *Model) Toggle() {
	m.show = !m.show
	if !m.show {
		m.focus = false
	}
}

func (m *Model) ToggleFocus() {
	m.focus = !m.focus
}

func (m *Model) Focus(focus bool) {
	m.focus = focus
}

func (m *Model) SyncState() {
	// Nothing to do
}

type browserListItem struct {
	entry library.Entry
}

func (i browserListItem) FilterValue() string {
	return i.entry.Name
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
