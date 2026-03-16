package playlist

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/charmbracelet/x/ansi"
)

var (
	volumeLabel  = "Volume:  "
	playingLabel = "Playing: "
)

type Model struct {
	width  int
	height int
	show   bool
	focus  bool
	app    *core.App
	list   list.Model
	volume *volumeModel
	status *statusModel

	playing   int
	playState core.PlaybackState
	rows      []playlistRow
	posWidth  int
}

func NewModel(appRef *core.App) *Model {
	m := &Model{
		app:    appRef,
		volume: newVolumeModel(volumeLabel, appRef),
		status: newStatusModel(playingLabel, appRef),
	}
	delegate := newPlaylistDelegate(m)
	playlistList := list.New(nil, delegate, 0, 0)
	playlistList.SetShowTitle(false)
	playlistList.SetShowFilter(false)
	playlistList.SetShowStatusBar(false)
	playlistList.SetShowPagination(true)
	playlistList.SetShowHelp(false)
	playlistList.SetFilteringEnabled(true)
	playlistList.DisableQuitKeybindings()
	playlistList.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup", "pageup"), key.WithHelp("pgup", "prev page"))
	playlistList.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown", "pagedown"), key.WithHelp("pgdn", "next page"))
	playlistList.KeyMap.GoToStart = key.NewBinding(key.WithKeys("home", "pos1"), key.WithHelp("home", "go to start"))
	playlistList.KeyMap.GoToEnd = key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "go to end"))
	playlistList.FilterInput.Prompt = "Search: "
	playlistList.FilterInput.Placeholder = "/"
	playlistList.FilterInput.SetStyles(textinput.Styles{
		Focused: textinput.StyleState{Text: styleSearchActive, Prompt: styleSearchActive},
		Blurred: textinput.StyleState{Text: styleSearchInactive, Prompt: styleSearchInactive, Placeholder: styleSearchInactive},
	})
	playlistList.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	playlistList.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("•")
	playlistList.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("·")
	playlistList.Paginator.ActiveDot = playlistList.Styles.ActivePaginationDot.String()
	playlistList.Paginator.InactiveDot = playlistList.Styles.InactivePaginationDot.String()
	m.list = playlistList
	return m
}

func (m *Model) UpdateSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.volume.UpdateSize(m.width - stylePanelUnfocused.GetHorizontalFrameSize())
	m.status.UpdateSize(m.width - stylePanelUnfocused.GetHorizontalFrameSize())
}

func (m *Model) View() string {
	state := m.app.State()
	status := m.status.View()
	volume := m.volume.View()

	title := styleTitle
	panelStyle := stylePanelUnfocused
	if m.focus {
		title = styleTitleFocused
		panelStyle = stylePanelFocused
	}
	panelStyle = panelStyle.Width(m.width).Height(m.height)
	innerWidth := max(0, m.width-panelStyle.GetHorizontalFrameSize())
	innerHeight := max(0, m.height-panelStyle.GetVerticalFrameSize())

	titleLine := title.Render("🎵 Playlist") + " (" + playStateStyle(state).Render(playStateLabel(state)) + ", " + styleStatusMeta.Render(queueModeLabel(state.QueueMode)) + ")"
	titleLine = ansi.Truncate(titleLine, innerWidth, "…")
	lines := []string{
		titleLine,
		m.searchView(),
		styleSeparator.Render(strings.Repeat("─", innerWidth)),
	}
	if state.PlaylistErr != nil {
		lines = append(lines, styleError.Render(state.PlaylistErr.Error()))
	}

	footerLines := 0
	if status != "" || volume != "" {
		footerLines = 1
		if status != "" {
			footerLines++
		}
		if volume != "" {
			footerLines++
		}
	}
	availableLines := max(0, innerHeight-len(lines)-footerLines)

	m.list.SetSize(innerWidth, availableLines)
	itemCount := len(m.list.Items())
	visibleCount := len(m.list.VisibleItems())
	contentLines := 0

	switch {
	case itemCount == 0 && availableLines > 0:
		lines = append(lines, styleEmpty.Render("(empty)"))
		contentLines = 1
	case availableLines > 0:
		if visibleCount == 0 && m.filterActive() {
			lines = append(lines, styleEmpty.Render("(no matches)"))
			contentLines = 1
		} else if visibleCount == 0 {
			lines = append(lines, styleEmpty.Render("(empty)"))
			contentLines = 1
		} else {
			lines = append(lines, m.list.View())
			// list.View() is sized to availableLines and already provides
			// the full content area height when rows are present.
			contentLines = availableLines
		}
	}
	for i := contentLines; i < availableLines; i++ {
		lines = append(lines, "")
	}

	if status != "" || volume != "" {
		lines = append(lines, styleSeparator.Render(strings.Repeat("─", innerWidth)))
		if status != "" {
			lines = append(lines, status)
		}
		if volume != "" {
			lines = append(lines, volume)
		}
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) ToggleFocus() {
	m.focus = !m.focus
}

func (m *Model) Focus(focus bool) {
	m.focus = focus
}

func (m *Model) Focused() bool {
	return m.focus
}

func (m *Model) Init() tea.Cmd {
	m.SyncState()
	return nil
}

func (m *Model) Show(show bool) {
	m.show = show
	if !show {
		m.focus = false
	}
}

func (m *Model) Shutdown() {
	// Nothing to do
}

func (m *Model) SyncState() {
	state := m.app.State()
	m.playing = state.Playing
	m.playState = state.PlayState
	m.updateItems(state)

	cursor := selectedTrack(state)
	if m.filterActive() {
		m.ensureFilteredSelection(state)
		return
	}
	if cursor >= 0 {
		m.selectTrack(cursor)
		return
	}
	if len(m.list.VisibleItems()) > 0 {
		m.list.Select(0)
	}
}

func (m *Model) HandleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.show {
		return nil, false
	}

	state := m.app.State()
	if handled, cmd := m.updateSearch(msg, state); handled {
		return cmd, true
	}
	if cmd, handled := m.updateNav(msg, state); handled {
		return cmd, true
	}

	if m.focus {
		switch msg.Key().Text {
		case "i":
			return ToggleTrackInfoCmd(), true
		case "L":
			return ToggleLyricsCmd(), true
		case "c":
			_ = m.app.Dispatch(core.Command{Type: core.CmdClear})
			return nil, true
		case "x":
			_ = m.app.Dispatch(core.Command{Type: core.CmdRemoveAt, Index: state.Cursor})
			return nil, true
		}
	}

	switch msg.String() {
	case "space":
		_ = m.app.Dispatch(core.Command{Type: core.CmdTogglePause})
		return nil, true
	}

	switch msg.Key().Text {
	case "n":
		_ = m.app.Dispatch(core.Command{Type: core.CmdNext})
		return nil, true
	case "p":
		_ = m.app.Dispatch(core.Command{Type: core.CmdPrev})
		return nil, true
	case "+":
		_ = m.app.Dispatch(core.Command{Type: core.CmdVolumeUp})
		return nil, true
	case "-":
		_ = m.app.Dispatch(core.Command{Type: core.CmdVolumeDown})
		return nil, true
	case "m":
		_ = m.app.Dispatch(core.Command{Type: core.CmdToggleMute})
		return nil, true
	case "s":
		_ = m.app.Dispatch(core.Command{Type: core.CmdStop})
		return nil, true
	case ",":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: -10 * time.Second})
		return nil, true
	case ".":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: 10 * time.Second})
		return nil, true
	case "<":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: -60 * time.Second})
		return nil, true
	case ">":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: 60 * time.Second})
		return nil, true
	case "M":
		mode := nextQueueMode(state.QueueMode)
		_ = m.app.Dispatch(core.Command{Type: core.CmdSetQueueMode, Mode: mode})
		return nil, true
	}
	return nil, false
}

func (m *Model) updateNav(msg tea.KeyMsg, state core.State) (tea.Cmd, bool) {
	if !m.focus {
		return nil, false
	}
	switch msg.String() {
	case "/", "up", "k", "down", "j", "pgup", "pageup", "pgdown", "pagedown", "home", "pos1", "end", "esc":
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if msg.String() == "/" {
			m.syncFilterWhileEditing()
		}
		m.syncSelectionFromList(state)
		return cmd, true
	case "enter":
		_ = m.app.Dispatch(core.Command{Type: core.CmdPlayFromCursor})
		return nil, true
	default:
		return nil, false
	}
}

func (m *Model) updateSearch(msg tea.KeyMsg, state core.State) (bool, tea.Cmd) {
	if !m.list.SettingFilter() {
		return false, nil
	}
	if msg.String() == "ctrl+c" {
		return false, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncFilterWhileEditing()
	m.syncSelectionFromList(state)
	return true, cmd
}

func (m *Model) syncFilterWhileEditing() {
	if !m.list.SettingFilter() {
		return
	}
	// The parent view forwards only key events, so apply filtering now.
	filter := m.list.FilterInput.Value()
	m.list.SetFilterText(filter)
	m.list.SetFilterState(list.Filtering)
}

func (m *Model) syncSelectionFromList(state core.State) {
	selected, ok := m.selectedTrack()
	if !ok {
		if m.filterActive() {
			m.ensureFilteredSelection(state)
		}
		return
	}
	if selected == selectedTrack(state) {
		return
	}
	_ = m.app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: selected})
}

func (m *Model) ensureFilteredSelection(state core.State) {
	cursor := selectedTrack(state)
	if cursor >= 0 && m.selectTrack(cursor) {
		return
	}

	first, ok := m.firstVisibleTrack()
	if !ok {
		return
	}
	m.list.Select(0)
	if first != cursor {
		_ = m.app.Dispatch(core.Command{Type: core.CmdSelectIndex, Index: first})
	}
}

func (m *Model) updateItems(state core.State) {
	m.posWidth = len(strconv.Itoa(max(1, len(state.Playlist))))

	nextRows := make([]playlistRow, 0, len(state.Playlist))
	for _, track := range state.Playlist {
		nextRows = append(nextRows, playlistRow{
			id:          track.ID,
			path:        track.Path,
			displayName: track.DisplayName(),
		})
	}
	if sameRows(m.rows, nextRows) {
		return
	}
	m.rows = nextRows

	items := make([]list.Item, 0, len(state.Playlist))
	for i, track := range state.Playlist {
		items = append(items, playlistListItem{
			index: i,
			track: track,
		})
	}
	if cmd := m.list.SetItems(items); cmd != nil {
		m.list, _ = m.list.Update(cmd())
	}
}

func (m *Model) selectedTrack() (int, bool) {
	item, ok := m.list.SelectedItem().(playlistListItem)
	if !ok {
		return -1, false
	}
	return item.index, true
}

func (m *Model) firstVisibleTrack() (int, bool) {
	items := m.list.VisibleItems()
	if len(items) == 0 {
		return -1, false
	}
	item, ok := items[0].(playlistListItem)
	if !ok {
		return -1, false
	}
	return item.index, true
}

func (m *Model) selectTrack(trackIndex int) bool {
	if trackIndex < 0 {
		return false
	}
	items := m.list.VisibleItems()
	for i, item := range items {
		playlistItem, ok := item.(playlistListItem)
		if !ok {
			continue
		}
		if playlistItem.index == trackIndex {
			m.list.Select(i)
			return true
		}
	}
	return false
}

func (m *Model) filterActive() bool {
	return m.list.SettingFilter() || m.list.IsFiltered()
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

func selectedTrack(state core.State) int {
	if state.Cursor != -1 {
		return state.Cursor
	}
	return state.Playing
}

type playlistRow struct {
	id          uint64
	path        string
	displayName string
}

func sameRows(a, b []playlistRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].id != b[i].id || a[i].path != b[i].path || a[i].displayName != b[i].displayName {
			return false
		}
	}
	return true
}

type playlistListItem struct {
	index int
	track core.Track
}

func (i playlistListItem) FilterValue() string {
	return i.track.DisplayName()
}

type playlistDelegate struct {
	model *Model
}

func newPlaylistDelegate(model *Model) list.ItemDelegate {
	return playlistDelegate{model: model}
}

func (playlistDelegate) Height() int {
	return 1
}

func (playlistDelegate) Spacing() int {
	return 0
}

func (playlistDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d playlistDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	playlistItem, ok := item.(playlistListItem)
	if !ok {
		return
	}
	name := playlistItem.track.DisplayName()
	prefix := "  "
	isPlaying := playlistItem.index == d.model.playing && d.model.playState == core.PlaybackPlaying
	if isPlaying {
		prefix = "▶︎ "
	}
	isPaused := playlistItem.index == d.model.playing && d.model.playState == core.PlaybackPaused
	if isPaused {
		prefix = "⏸ "
	}
	pos := fmt.Sprintf("%*d ", max(1, d.model.posWidth), playlistItem.index+1)
	line := ansi.Truncate(prefix+pos+name, max(0, m.Width()), "…")
	if isPlaying {
		line = stylePlaying.Render(line)
	}
	if isPaused {
		line = stylePaused.Render(line)
	}
	isSelected := index == m.Index()
	if isSelected {
		line = styleSelected.Render(line)
	}
	_, _ = fmt.Fprint(w, line)
}

func playStateLabel(state core.State) string {
	switch state.PlayState {
	case core.PlaybackPaused:
		return "paused"
	case core.PlaybackPlaying:
		return "playing"
	default:
		return "stopped"
	}
}

func nextQueueMode(mode core.QueueMode) core.QueueMode {
	switch mode {
	case core.QueueModeLinear:
		return core.QueueModeShuffle
	case core.QueueModeShuffle:
		return core.QueueModeRepeatOne
	case core.QueueModeRepeatOne:
		return core.QueueModeRepeatAll
	case core.QueueModeRepeatAll:
		return core.QueueModeStopAfterCurrent
	default:
		return core.QueueModeLinear
	}
}

func queueModeLabel(mode core.QueueMode) string {
	switch mode {
	case core.QueueModeShuffle:
		return "shuffle"
	case core.QueueModeRepeatOne:
		return "repeat-one"
	case core.QueueModeRepeatAll:
		return "repeat-all"
	case core.QueueModeStopAfterCurrent:
		return "stop-after-current"
	default:
		return "linear"
	}
}
