package footer

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist/spectrum"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist/status"
	"github.com/bpicode/tmus/internal/ui/view/home/playlist/volume"
)

const (
	labelWidth    = 10
	spectrumLabel = "Spectrum:"
	volumeLabel   = "Volume:"
	playingLabel  = "Playing:"
)

// Model composes the playback details shown below the playlist.
type Model struct {
	width    int
	layout   footerLayout
	spectrum *spectrum.Model
	status   *status.Model
	volume   *volume.Model
}

type footerLayout struct {
	height       int
	spectrumRows int
	showStatus   bool
	showVolume   bool
}

// Config contains the dependencies of a footer Model.
type Config struct {
	Theme theme.Theme
	App   *core.App
	FPS   int
}

// NewModel creates a playlist footer and its nested models.
func NewModel(cfg Config) *Model {
	return &Model{
		spectrum: spectrum.NewModel(formatLabel(spectrumLabel), cfg.App, cfg.FPS, cfg.Theme),
		status:   status.NewModel(formatLabel(playingLabel), cfg.App, cfg.Theme),
		volume:   volume.NewModel(formatLabel(volumeLabel), cfg.App, cfg.Theme),
	}
}

// Init initializes the nested footer models.
func (m *Model) Init() tea.Cmd {
	return m.spectrum.Init()
}

// Update forwards messages to the nested footer models.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spectrum, cmd = m.spectrum.Update(msg)
	return m, cmd
}

// SetSize sets the width and maximum height available to the footer content.
func (m *Model) SetSize(width, height int) {
	m.width = max(0, width)
	m.spectrum.UpdateSize(m.width)
	m.status.UpdateSize(m.width)
	m.volume.UpdateSize(m.width)

	m.layout = m.calculateLayout(max(0, height), m.status.Height(), m.volume.Height())
}

// Height returns the number of rows occupied by the footer content.
func (m *Model) Height() int {
	return m.layout.height
}

// View renders the footer content.
func (m *Model) View() string {
	if m.layout.height == 0 {
		return ""
	}

	lines := make([]string, 0, m.layout.height)
	if m.layout.spectrumRows > 0 {
		lines = append(lines, strings.Split(m.spectrum.View(m.layout.spectrumRows), "\n")...)
	}
	if m.layout.showStatus {
		lines = append(lines, m.status.View())
	}
	if m.layout.showVolume {
		lines = append(lines, m.volume.View())
	}
	return strings.Join(lines, "\n")
}

func (m *Model) calculateLayout(availableHeight, statusHeight, volumeHeight int) footerLayout {
	var layout footerLayout
	remainingHeight := availableHeight
	if statusHeight > 0 && statusHeight <= remainingHeight {
		layout.showStatus = true
		layout.height += statusHeight
		remainingHeight -= statusHeight
	}
	if volumeHeight > 0 && volumeHeight <= remainingHeight {
		layout.showVolume = true
		layout.height += volumeHeight
		remainingHeight -= volumeHeight
	}

	if m.width > labelWidth {
		layout.spectrumRows = spectrumRowCount(remainingHeight)
	}
	layout.height += layout.spectrumRows
	return layout
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

func formatLabel(label string) string {
	return fmt.Sprintf("%-*s", labelWidth, label)
}
