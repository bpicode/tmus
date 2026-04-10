package track_info

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/ui/components/terminalimage"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
	"github.com/bpicode/tmus/internal/ui/theme"
)

const defaultArtworkAspect = 2.0

type Model struct {
	show      bool
	width     int
	height    int
	trackID   uint64
	trackPath string
	loading   bool
	err       string
	viewport  viewport.Model
	artwork   terminalimage.Model
	data      library.Metadata
	app       *core.App
	styles    styles
}

type Config struct {
	Theme         theme.Theme
	ArtworkAspect float64
	App           *core.App
}

func NewModel(cfg Config) *Model {
	artworkAspect := cfg.ArtworkAspect
	if artworkAspect <= 0 {
		artworkAspect = defaultArtworkAspect
	}
	vp := viewport.New()
	vp.LeftGutterFunc = viewport.NoGutter
	styles := newStyles(cfg.Theme)
	return &Model{
		viewport: vp,
		artwork:  terminalimage.NewModel(artworkAspect, "No artwork"),
		app:      cfg.App,
		styles:   styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleSizeMsg(msg)
	case tea.KeyPressMsg:
		return m.handleKeyPressMsg(msg)
	case core.MetadataEvent:
		return m.handleMetadataEvent(msg)
	default:
		return m, nil, false
	}
}

func (m *Model) handleSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd, bool) {
	m.width = msg.Width
	m.height = msg.Height
	return m, nil, false
}

func (m *Model) handleMetadataEvent(event core.MetadataEvent) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}
	if event.TrackID != m.trackID {
		return m, nil, false
	}
	if event.Path != m.trackPath {
		return m, nil, false
	}
	if event.Scope != core.MetadataExtended {
		return m, nil, false
	}
	m.loading = false
	if event.Err != nil {
		m.err = event.Err.Error()
		return m, nil, false
	}
	m.err = ""
	m.data = event.Metadata
	return m, nil, false
}

func (m *Model) handleKeyPressMsg(msg tea.KeyPressMsg) (*Model, tea.Cmd, bool) {
	if !m.show {
		return m, nil, false
	}
	switch msg.String() {
	case "q", "esc", "i":
		m.Show(false)
		return m, nil, true
	case "up", "k":
		m.viewport.ScrollUp(1)
		return m, nil, true
	case "down", "j":
		m.viewport.ScrollDown(1)
		return m, nil, true
	case "pgup", "pageup":
		m.viewport.PageUp()
		return m, nil, true
	case "pgdown", "pagedown":
		m.viewport.PageDown()
		return m, nil, true
	case "home", "pos1":
		m.viewport.GotoTop()
		return m, nil, true
	case "end":
		m.viewport.GotoBottom()
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m *Model) Show(show bool) {
	if show {
		state := m.app.State()
		if len(state.Playlist) == 0 {
			return
		}
		cursor := state.Playing
		if state.Cursor != -1 {
			cursor = state.Cursor
		}
		if cursor < 0 || cursor >= len(state.Playlist) {
			return
		}
		track := state.Playlist[cursor]
		if track.ID == 0 || track.Path == "" {
			return
		}

		m.show = true
		m.trackID = track.ID
		m.trackPath = track.Path
		m.loading = true
		m.err = ""
		m.viewport.GotoTop()
		m.data = library.Metadata{}
		_ = m.app.Dispatch(core.Command{
			Type:    core.CmdRequestMetadata,
			TrackID: track.ID,
			Path:    track.Path,
			Scope:   core.MetadataExtended,
		})
	} else {
		m.show = false
		m.trackID = 0
		m.trackPath = ""
		m.loading = false
		m.err = ""
		m.viewport.GotoTop()
		m.data = library.Metadata{}
	}
}

func (m *Model) Visible() bool {
	return m.show
}

func (m *Model) View() string {
	if m.width < 1 || m.height < 1 {
		return ""
	}
	contentWidth, contentHeight := m.innerSize()
	if contentHeight < 1 || contentWidth < 1 {
		return m.styles.Overlay.Width(m.width).Height(m.height).Render("")
	}

	header := m.headerLines(contentWidth)
	headerHeight := min(len(header), contentHeight)
	header = header[:headerHeight]
	headerBlock := lipgloss.NewStyle().Width(contentWidth).Height(headerHeight).Render(strings.Join(header, "\n"))
	if contentHeight <= headerHeight {
		return m.styles.Overlay.Width(m.width).Height(m.height).Render(headerBlock)
	}

	areaHeight := contentHeight - headerHeight
	fields := metadataFields(m.data)
	leftWidth, rightWidth, gap := metadataColumnWidths(contentWidth, areaHeight, fields, m.artwork.Aspect())

	rightLines := m.rightLines(rightWidth, fields)
	m.viewport.SetWidth(max(rightWidth, 0))
	m.viewport.SetHeight(max(areaHeight, 0))
	m.viewport.SetContentLines([]string{truncate.Right{}.MaxWidth(rightWidth).Render(rightLines)})

	styleRight := m.styles.Artwork.Width(leftWidth).Height(areaHeight)
	artworkWidth := leftWidth - m.styles.Artwork.GetHorizontalFrameSize()
	artworkHeight := areaHeight - m.styles.Artwork.GetVerticalFrameSize()
	m.artwork.SetSize(artworkWidth, artworkHeight)
	if m.data.Picture != nil {
		m.artwork.SetImage(&terminalimage.Data{Bytes: m.data.Picture.Data})
	} else {
		m.artwork.SetImage(nil)
	}

	leftBlock := ""
	if leftWidth > 0 {
		leftBlock = styleRight.Render(m.artwork.View())
	}
	rightBlock := ""
	if rightWidth > 0 {
		rightBlock = lipgloss.NewStyle().Width(rightWidth).Height(areaHeight).Render(m.viewport.View())
	}
	gapBlock := ""
	if gap > 0 {
		gapBlock = lipgloss.NewStyle().Width(gap).Height(areaHeight).Render("")
	}

	body := leftBlock
	if rightWidth > 0 {
		if gapBlock != "" {
			body = lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, gapBlock, rightBlock)
		} else {
			body = lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock)
		}
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, headerBlock, body)
	return m.styles.Overlay.Width(m.width).Height(m.height).Render(inner)
}

func (m *Model) innerSize() (int, int) {
	contentWidth := max(m.width-m.styles.Overlay.GetHorizontalFrameSize(), 0)
	contentHeight := max(m.height-m.styles.Overlay.GetVerticalFrameSize(), 0)
	return contentWidth, contentHeight
}

func (m *Model) headerLines(maxWidth int) []string {
	lines := make([]string, 0, 3)
	lines = append(lines, truncate.Right{Style: m.styles.Title}.MaxWidth(maxWidth).Render("🎵 Track info"))
	if m.trackPath != "" {
		lines = append(lines, truncate.Right{Style: m.styles.Subtitle}.MaxWidth(maxWidth).Render(m.trackPath))
	}
	lines = append(lines, "")
	return lines
}

func (m *Model) rightLines(maxWidth int, fields []field) string {
	if maxWidth < 1 {
		return ""
	}
	switch {
	case m.loading:
		return m.styles.Subtitle.Render("Loading...")
	case m.err != "":
		return m.styles.Error.Render(m.err)
	default:
		return m.metadataFieldLines(fields)
	}
}

func formatYear(year int) string {
	if year == 0 {
		return ""
	}
	return strconv.Itoa(year)
}

func formatPicture(pic *library.Picture) string {
	if pic == nil {
		return ""
	}
	label := pic.MIMEType
	if label == "" {
		label = "embedded"
	}
	if pic.Description != "" {
		label = label + " - " + pic.Description
	}
	return fmt.Sprintf("%s (%d bytes)", label, len(pic.Data))
}

func normalizeMetadataValue(value string) string {
	if value == "" {
		return "-"
	}
	return strings.Join(strings.Fields(value), " ")
}

type field struct {
	label string
	value string
}

func metadataFields(meta library.Metadata) []field {
	return []field{
		{label: "Title", value: meta.Title},
		{label: "Artist", value: meta.Artist},
		{label: "Album", value: meta.Album},
		{label: "Album artist", value: meta.AlbumArtist},
		{label: "Composer", value: meta.Composer},
		{label: "Genre", value: meta.Genre},
		{label: "Year", value: formatYear(meta.Year)},
		{label: "Picture", value: formatPicture(meta.Picture)},
	}
}

func (m *Model) metadataFieldLines(fields []field) string {
	maxKW := maxKeyWidth(fields)
	lines := make([]string, 0, len(fields))
	for _, f := range fields {
		labelPadding := strings.Repeat(" ", max(maxKW-len(f.label), 0))
		label := fmt.Sprintf("%s:%s", f.label, labelPadding)
		label = m.styles.MetadataKey.Render(label)
		value := normalizeMetadataValue(f.value)
		line := fmt.Sprintf("%s %s", label, value)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func maxKeyWidth(fields []field) int {
	maxKW := 0
	for _, f := range fields {
		label := f.label + ":"
		width := lipgloss.Width(label)
		if width > maxKW {
			maxKW = width
		}
	}
	return maxKW
}

func metadataColumnWidths(contentWidth, areaHeight int, fields []field, aspect float64) (leftWidth, rightWidth, gap int) {
	if contentWidth < 1 {
		return 0, 0, 0
	}
	gap = 2
	if contentWidth <= gap+1 {
		return contentWidth, 0, 0
	}
	available := contentWidth - gap
	leftMin := 10
	minValue := 8
	rightMin := max(20, maxKeyWidth(fields)+1+minValue)

	leftWidth = max(available-rightMin, leftMin)
	if areaHeight > 0 {
		maxLeft := max(1, int(float64(areaHeight)*aspect))
		leftWidth = min(leftWidth, maxLeft)
	}
	if leftWidth > available-1 {
		leftWidth = available - 1
	}
	if leftWidth < 1 {
		leftWidth = 0
		rightWidth = contentWidth
		gap = 0
		return leftWidth, rightWidth, gap
	}

	rightWidth = available - leftWidth
	if rightWidth < 1 {
		rightWidth = 1
		leftWidth = available - rightWidth
	}
	return leftWidth, rightWidth, gap
}
