package footer

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	width     int
	separator lipgloss.Style
	spectrum  *spectrum.Model
	status    *status.Model
	volume    *volume.Model
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
		separator: lipgloss.NewStyle().Foreground(cfg.Theme.Muted),
		spectrum:  spectrum.NewModel(formatLabel(spectrumLabel), cfg.App, cfg.FPS, cfg.Theme),
		status:    status.NewModel(formatLabel(playingLabel), cfg.App, cfg.Theme),
		volume:    volume.NewModel(formatLabel(volumeLabel), cfg.App, cfg.Theme),
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

// UpdateSize sets the number of terminal cells available to the footer.
func (m *Model) UpdateSize(width int) {
	m.width = max(0, width)
	m.spectrum.UpdateSize(m.width)
	m.status.UpdateSize(m.width)
	m.volume.UpdateSize(m.width)
}

// View renders the footer within the height left below the playlist header.
func (m *Model) View(availableHeight int) string {
	statusView := m.status.View()
	volumeView := m.volume.View()
	spectrumRows, footerLines := m.layout(availableHeight, statusView, volumeView)
	if footerLines == 0 {
		return ""
	}

	lines := []string{m.separator.Render(strings.Repeat("─", m.width))}
	if spectrumRows > 0 {
		lines = append(lines, strings.Split(m.spectrum.View(spectrumRows), "\n")...)
	}
	if statusView != "" {
		lines = append(lines, statusView)
	}
	if volumeView != "" {
		lines = append(lines, volumeView)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) layout(availableHeight int, statusView, volumeView string) (spectrumRows, footerLines int) {
	hasDetails := statusView != "" || volumeView != ""
	if hasDetails {
		footerLines = 1 // Separator.
		if statusView != "" {
			footerLines++
		}
		if volumeView != "" {
			footerLines++
		}
	}

	contentCapacity := max(0, availableHeight-footerLines)
	if !hasDetails && contentCapacity > 0 {
		contentCapacity-- // Reserve a separator if the spectrum fits.
	}
	if m.width > labelWidth {
		spectrumRows = spectrumRowCount(contentCapacity)
	}
	if hasDetails {
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

func formatLabel(label string) string {
	return fmt.Sprintf("%-*s", labelWidth, label)
}
