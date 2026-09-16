package playlist

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/components/sanitize"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist/spectrum"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist/status"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist/volume"
)

const (
	footerLabelWidth = 10
	spectrumLabel    = "Spectrum:"
	volumeLabel      = "Volume:"
	playingLabel     = "Playing:"
)

type Model struct {
	width    int
	height   int
	show     bool
	focus    bool
	app      *core.App
	list     list.Model
	volume   *volume.Model
	status   *status.Model
	spectrum *spectrum.Model

	playing   int
	playState core.PlaybackState
	rows      []playlistRow
	posWidth  int
	styles    styles
}

type Config struct {
	Theme theme.Theme
	App   *core.App
	FPS   int
}

func NewModel(cfg Config) *Model {
	styles := newStyles(cfg.Theme)
	m := &Model{
		app:      cfg.App,
		volume:   volume.NewModel(formatFooterLabel(volumeLabel), cfg.App, cfg.Theme),
		status:   status.NewModel(formatFooterLabel(playingLabel), cfg.App, cfg.Theme),
		spectrum: spectrum.NewModel(cfg.App, cfg.FPS, cfg.Theme),
		styles:   styles,
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
		Focused: textinput.StyleState{Text: styles.searchActive, Prompt: styles.searchActive},
		Blurred: textinput.StyleState{Text: styles.searchInactive, Prompt: styles.searchInactive, Placeholder: styles.searchInactive},
		Cursor:  textinput.CursorStyle{Blink: true},
	})
	playlistList.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	playlistList.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("•")
	playlistList.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).SetString("·")
	playlistList.Paginator.ActiveDot = playlistList.Styles.ActivePaginationDot.String()
	playlistList.Paginator.InactiveDot = playlistList.Styles.InactivePaginationDot.String()
	m.list = playlistList
	return m
}

func (m *Model) Init() tea.Cmd {
	_, cmd, _ := m.syncState()
	return tea.Batch(cmd, m.spectrum.Init())
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd, bool) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.spectrum, cmd = m.spectrum.Update(msg)
	cmds = append(cmds, cmd)

	var stop bool
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m, cmd, stop = m.handleSizeMsg(msg)
	case tea.KeyPressMsg:
		m, cmd, stop = m.handleKeyPressMsg(msg)
	case core.StateEvent:
		m, cmd, stop = m.syncState()
	case core.MetadataEvent:
		m, cmd, stop = m.syncState()
	default:
		m, cmd, stop = m.handleRemaining(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...), stop
}

func (m *Model) handleSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd, bool) {
	m.width = msg.Width
	m.height = msg.Height
	m.volume.UpdateSize(m.width - m.styles.panelUnfocused.GetHorizontalFrameSize())
	m.status.UpdateSize(m.width - m.styles.panelUnfocused.GetHorizontalFrameSize())
	return m, nil, false
}

func (m *Model) handleKeyPressMsg(msg tea.KeyMsg) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}

	state := m.app.State()
	if handled, cmd := m.updateSearch(msg, state); handled {
		return m, cmd, true
	}
	if cmd, handled := m.updateNav(msg, state); handled {
		return m, cmd, true
	}

	if m.focus {
		switch msg.String() {
		case "i":
			return m, toggleTrackInfoCmd(), true
		case "L":
			return m, toggleLyricsCmd(), true
		case "c":
			_ = m.app.Dispatch(core.Command{Type: core.CmdClear})
			return m, nil, true
		case "x", "delete":
			_ = m.app.Dispatch(core.Command{Type: core.CmdRemoveAt, Index: state.Cursor})
			return m, nil, true
		}
	}

	switch msg.String() {
	case "space":
		_ = m.app.Dispatch(core.Command{Type: core.CmdTogglePause})
		return m, nil, true
	}

	switch msg.Key().Text {
	case "n":
		_ = m.app.Dispatch(core.Command{Type: core.CmdNext})
		return m, nil, true
	case "p":
		_ = m.app.Dispatch(core.Command{Type: core.CmdPrev})
		return m, nil, true
	case "+":
		_ = m.app.Dispatch(core.Command{Type: core.CmdVolumeUp})
		return m, nil, true
	case "-":
		_ = m.app.Dispatch(core.Command{Type: core.CmdVolumeDown})
		return m, nil, true
	case "m":
		_ = m.app.Dispatch(core.Command{Type: core.CmdToggleMute})
		return m, nil, true
	case "s":
		_ = m.app.Dispatch(core.Command{Type: core.CmdStop})
		return m, nil, true
	case ",":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: -10 * time.Second})
		return m, nil, true
	case ".":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: 10 * time.Second})
		return m, nil, true
	case "<":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: -60 * time.Second})
		return m, nil, true
	case ">":
		_ = m.app.Dispatch(core.Command{Type: core.CmdSeekBy, Offset: 60 * time.Second})
		return m, nil, true
	case "M":
		mode := nextQueueMode(state.QueueMode)
		_ = m.app.Dispatch(core.Command{Type: core.CmdSetQueueMode, Mode: mode})
		return m, nil, true
	default:
		return m.handleRemaining(msg)
	}
}

func (m *Model) handleRemaining(msg tea.Msg) (*Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd, false
}

func (m *Model) View() string {
	state := m.app.State()
	statusView := m.status.View()
	volumeView := m.volume.View()

	title := m.styles.titleUnfocused
	panelStyle := m.styles.panelUnfocused
	if m.focus {
		title = m.styles.titleFocused
		panelStyle = m.styles.panelFocused
	}
	panelStyle = panelStyle.Width(m.width).Height(m.height)
	innerWidth := max(0, m.width-panelStyle.GetHorizontalFrameSize())
	innerHeight := max(0, m.height-panelStyle.GetVerticalFrameSize())

	titleLine := fmt.Sprintf("%s (%s, %s)", title.Render("🎵 Playlist"), m.styles.playStateStyle(state).Render(playStateLabel(state)), m.styles.statusQueueMode.Render(queueModeLabel(state.QueueMode)))
	titleLine = truncate.Right{}.MaxWidth(innerWidth).Render(titleLine)

	lines := []string{
		titleLine,
		m.searchView(),
		m.styles.separator.Render(strings.Repeat("─", innerWidth)),
	}
	if state.PlaylistErr != nil {
		lines = append(lines, m.styles.err.Render(sanitize.TerminalText(state.PlaylistErr.Error())))
	}

	hasFooterDetails := statusView != "" || volumeView != ""
	spectrumRows, footerLines := spectrumLayout(innerWidth, innerHeight, len(lines), statusView, volumeView)
	availableLines := max(0, innerHeight-len(lines)-footerLines)

	m.list.SetSize(innerWidth, availableLines)
	itemCount := len(m.list.Items())
	visibleCount := len(m.list.VisibleItems())
	contentLines := 0

	switch {
	case itemCount == 0 && availableLines > 0:
		lines = append(lines, m.styles.empty.Render("(empty)"))
		contentLines = 1
	case availableLines > 0:
		if visibleCount == 0 && m.filterActive() {
			lines = append(lines, m.styles.empty.Render("(no matches)"))
			contentLines = 1
		} else if visibleCount == 0 {
			lines = append(lines, m.styles.empty.Render("(empty)"))
			contentLines = 1
		} else {
			lines = append(lines, m.list.View())
			// list.View() is sized to availableLines and already provides
			// the full content area height when rows are present.
			contentLines = availableLines
		}
	}
	for range availableLines - contentLines {
		lines = append(lines, "")
	}

	if hasFooterDetails || spectrumRows > 0 {
		lines = append(lines, m.styles.separator.Render(strings.Repeat("─", innerWidth)))
		if spectrumRows > 0 {
			lines = append(lines, m.spectrumView(innerWidth, spectrumRows)...)
		}
		if statusView != "" {
			lines = append(lines, statusView)
		}
		if volumeView != "" {
			lines = append(lines, volumeView)
		}
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func formatFooterLabel(label string) string {
	return fmt.Sprintf("%-*s", footerLabelWidth, label)
}

func (m *Model) spectrumView(width, rows int) []string {
	contentWidth := max(0, width-footerLabelWidth)
	lines := strings.Split(m.spectrum.View(contentWidth, rows), "\n")
	for row := range lines {
		label := strings.Repeat(" ", footerLabelWidth)
		if row == len(lines)-1 {
			label = formatFooterLabel(spectrumLabel)
		}
		lines[row] = label + lines[row]
	}
	return lines
}

func spectrumLayout(innerWidth, innerHeight, headerLines int, status, volume string) (spectrumRows, footerLines int) {
	hasFooterDetails := status != "" || volume != ""
	if hasFooterDetails {
		footerLines = 1 // Separator.
		if status != "" {
			footerLines++
		}
		if volume != "" {
			footerLines++
		}
	}

	contentCapacity := max(0, innerHeight-headerLines-footerLines)
	if !hasFooterDetails && contentCapacity > 0 {
		contentCapacity-- // Reserve a separator if the spectrum fits.
	}
	if innerWidth > footerLabelWidth {
		spectrumRows = spectrumRowCount(contentCapacity)
	}
	if hasFooterDetails {
		footerLines += spectrumRows
	} else if spectrumRows > 0 {
		footerLines = spectrumRows + 1
	}
	return spectrumRows, footerLines
}

func spectrumRowCount(contentCapacity int) int {
	switch {
	case contentCapacity >= 3:
		return 2
	case contentCapacity >= 2:
		return 1
	default:
		return 0
	}
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

func (m *Model) Searching() bool {
	return m.show && m.focus && m.list.SettingFilter()
}

func (m *Model) Show(show bool) {
	m.show = show
	if !show {
		m.focus = false
	}
}

func (m *Model) syncState() (*Model, tea.Cmd, bool) {
	state := m.app.State()
	m.playing = state.Playing
	m.playState = state.Playback.State
	m.updateItems(state)

	cursor := selectedTrack(state)
	if m.filterActive() {
		m.ensureFilteredSelection(state)
		return m, nil, false
	}
	if cursor >= 0 {
		m.selectTrack(cursor)
		return m, nil, false
	}
	if len(m.list.VisibleItems()) > 0 {
		m.list.Select(0)
	}
	return m, nil, false
}

func (m *Model) updateNav(msg tea.KeyMsg, state core.State) (tea.Cmd, bool) {
	if !m.focus {
		return nil, false
	}
	switch msg.String() {
	case "/", "up", "k", "down", "j", "pgup", "pageup", "pgdown", "pagedown", "home", "pos1", "end", "esc":
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		m.syncSelectionFromList(state)
		return cmd, true
	case "alt+up", "alt+k":
		_ = m.app.Dispatch(core.Command{Type: core.CmdMoveUp})
		return nil, true
	case "alt+down", "alt+j":
		_ = m.app.Dispatch(core.Command{Type: core.CmdMoveDown})
		return nil, true
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
	m.syncSelectionFromList(state)
	return true, cmd
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
	if slices.Equal(m.rows, nextRows) {
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
		return m.styles.searchActive.Render("Search: " + sanitize.TerminalText(m.list.FilterValue()))
	default:
		return m.styles.searchInactive.Render("Search: /")
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

type playlistListItem struct {
	index int
	track core.Track
}

func (i playlistListItem) FilterValue() string {
	return sanitize.TerminalText(i.track.DisplayName())
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
	name := sanitize.TerminalText(playlistItem.track.DisplayName())
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
	style := lipgloss.NewStyle()
	if isPlaying {
		style = style.Inherit(d.model.styles.playing)
	}
	if isPaused {
		style = style.Inherit(d.model.styles.paused)
	}
	isSelected := index == m.Index()
	if isSelected {
		style = style.Inherit(d.model.styles.selected)
	}
	truncateRight := truncate.Right{Style: style}.MaxWidth(m.Width())
	_, _ = fmt.Fprint(w, truncateRight.Render(prefix+pos+name))
}

func playStateLabel(state core.State) string {
	switch state.Playback.State {
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
