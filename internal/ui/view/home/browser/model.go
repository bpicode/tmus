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
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/ui/components/errorview"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type Model struct {
	Cwd        string
	homeDir    string
	entries    []library.Entry
	showHidden bool
	width      int
	height     int
	show       bool
	focus      bool
	app        *core.App
	list       list.Model
	errorView  *errorview.Model
	styles     styles
}

type Config struct {
	Cwd     string
	HomeDir string
	Theme   theme.Theme
	App     *core.App
}

func NewModel(cfg Config) *Model {
	styles := newStyles(cfg.Theme)
	delegate := newEntryDelegate(styles)
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
		Focused: textinput.StyleState{Text: styles.searchActive, Prompt: styles.searchActive},
		Blurred: textinput.StyleState{Text: styles.searchInactive, Prompt: styles.searchInactive, Placeholder: styles.searchInactive},
		Cursor:  textinput.CursorStyle{Blink: true},
	})
	browserList.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	browserList.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("•")
	browserList.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("·")
	browserList.Paginator.ActiveDot = browserList.Styles.ActivePaginationDot.String()
	browserList.Paginator.InactiveDot = browserList.Styles.InactivePaginationDot.String()

	b := &Model{
		Cwd:       cfg.Cwd,
		homeDir:   cfg.HomeDir,
		app:       cfg.App,
		list:      browserList,
		errorView: errorview.New(errorview.Styles{Error: styles.err}),
		styles:    styles,
	}
	return b
}

func (m *Model) loadDir(path string) tea.Cmd {
	m.Cwd = path
	return loadDirCmd(path, m.showHidden)
}

func (m *Model) loadHomeDir() tea.Cmd {
	if m.homeDir != "" {
		return m.loadDir(m.homeDir)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		m.errorView.SetErr(err)
		return nil
	}
	return m.loadDir(homeDir)
}

func (m *Model) openSelection() tea.Cmd {
	m.errorView.SetErr(nil)
	selected, ok := m.selected()
	if !ok || !selected.IsDir {
		if ok && !library.IsArchivePath(selected.Path) {
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
	m.errorView.SetErr(nil)
	if library.IsArchivePath(m.Cwd) {
		scheme, archivePath, inner, err := library.SplitArchivePath(m.Cwd)
		if err != nil {
			m.errorView.SetErr(err)
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
		return m.loadDir(library.BuildArchivePath(scheme, archivePath, parent))
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
	return loadDirCmd(m.Cwd, m.showHidden)
}

func (m *Model) View() string {
	var sb strings.Builder
	title := m.styles.titleUnfocused
	panelStyle := m.styles.panelUnfocused.Width(m.width).Height(m.height)
	if m.focus {
		title = m.styles.titleFocused
		panelStyle = m.styles.panelFocused.Width(m.width).Height(m.height)
	}
	sb.WriteString(title.Render("📂 Files"))
	sb.WriteString("\n")
	pathWidth := max(0, m.width-panelStyle.GetHorizontalFrameSize())
	sb.WriteString(m.styles.cwd.MaxWidth(pathWidth).Render(m.Cwd))
	sb.WriteString("\n")
	sb.WriteString(m.searchView())
	sb.WriteString("\n")
	sb.WriteString(m.styles.separator.Render(strings.Repeat("─", max(0, m.width-panelStyle.GetHorizontalFrameSize()))))
	sb.WriteString("\n")

	if m.errorView.HasErr() {
		sb.WriteString(m.errorView.View())
		return panelStyle.Render(sb.String())
	}

	if len(m.entries) == 0 {
		sb.WriteString(m.styles.empty.Render("(empty)"))
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

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleSizeMsg(msg)
	case tea.KeyPressMsg:
		return m.handleKeyPressMsg(msg)
	case loadDirMsg:
		return m.handleLoadDirMsg(msg)
	default:
		return m.handleRemaining(msg)
	}
}

func (m *Model) handleRemaining(msg tea.Msg) (*Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd, false
}

func (m *Model) handleSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd, bool) {
	m.height = msg.Height
	m.width = msg.Width
	m.list.SetSize(max(0, m.width-m.styles.panelFocused.GetHorizontalFrameSize()), max(0, m.height-6))
	return m, nil, false
}

func (m *Model) handleKeyPressMsg(msg tea.KeyMsg) (*Model, tea.Cmd, bool) {
	if !m.show || !m.focus {
		return m, nil, false
	}
	if handled, cmd := m.updateSearch(msg); handled {
		return m, cmd, true
	}
	if cmd, handled := m.updateNav(msg); handled {
		return m, cmd, true
	}
	return m, nil, false
}

func (m *Model) handleLoadDirMsg(msg loadDirMsg) (*Model, tea.Cmd, bool) {
	if m.Cwd != msg.Path {
		// This can happen when the user quickly navigates to another directory before the previous one finishes loading.
		// In this case we ignore the message since it's outdated.
		return m, nil, false
	}

	if msg.Err != nil {
		m.errorView.SetErr(msg.Err)
		return m, nil, false
	}

	prevIndex := m.list.Index()
	m.entries = msg.Items
	m.updateListItems(prevIndex)
	m.errorView.SetErr(nil)
	return m, nil, false
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
	return true, cmd
}

func (m *Model) clearSearch() {
	m.list.ResetFilter()
}

func (m *Model) searchView() string {
	switch {
	case m.list.SettingFilter():
		return m.list.FilterInput.View()
	case m.list.IsFiltered():
		return m.styles.searchActive.Render("Search: " + m.list.FilterValue())
	default:
		return m.styles.searchInactive.Render("Search: /")
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
		m.errorView.SetErr(nil)
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

func (m *Model) Searching() bool {
	return m.show && m.focus && m.list.SettingFilter()
}

func (m *Model) ToggleFocus() {
	m.focus = !m.focus
}

func (m *Model) Focus(focus bool) {
	m.focus = focus
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
