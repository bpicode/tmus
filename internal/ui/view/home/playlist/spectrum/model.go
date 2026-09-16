package spectrum

import (
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/theme"
)

const (
	attackDuration   = 40 * time.Millisecond
	releaseDuration  = 200 * time.Millisecond
	displayFloorDB   = -60.0
	settledLevel     = 0.01
	energyColorSteps = 17
)

var barLevels = [...]rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Model smooths spectrum snapshots and renders them as vertical bars.
// It also tracks snapshot identity so seeks, track changes, and pause/resume
// transitions cannot reuse stale measurements.
type Model struct {
	app   *core.App
	fps   int
	width int

	displayed [core.SpectrumBandCount]float64
	targets   [core.SpectrumBandCount]float64

	playbackID uint64
	generation uint64
	sequence   uint64
	lastUpdate time.Time
	playing    bool

	energyStyles   [energyColorSteps]lipgloss.Style
	tickGeneration uint64
}

// NewModel creates a spectrum visualizer whose bar colors blend from the theme's
// primary accent at low energy to its secondary accent at high energy.
func NewModel(app *core.App, fps int, th theme.Theme) *Model {
	m := &Model{app: app, fps: fps}
	for i, color := range lipgloss.Blend1D(energyColorSteps, th.Primary, th.Secondary) {
		m.energyStyles[i] = lipgloss.NewStyle().Foreground(color)
	}
	return m
}

// Init starts animation when playback is already active.
func (m *Model) Init() tea.Cmd {
	if m.app == nil || m.app.State().Playback.State != core.PlaybackPlaying {
		return nil
	}
	return m.scheduleTick(0)
}

// Update handles visualizer messages and returns any follow-up animation tick.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, m.handleTick(msg)
	case core.StateEvent:
		if msg.Changes&(core.StateChangePlaying|core.StateChangePlayback) != 0 {
			return m, m.scheduleTick(0)
		}
	}
	return m, nil
}

// UpdateSize sets the number of terminal cells available for spectrum bars.
func (m *Model) UpdateSize(width int) {
	m.width = max(0, width)
}

type tickMsg struct {
	at         time.Time
	generation uint64
}

func (m *Model) handleTick(msg tickMsg) tea.Cmd {
	if msg.generation != m.tickGeneration || m.app == nil {
		return nil
	}
	state := m.app.State()
	m.advance(m.app.Spectrum(), state.Playback.State == core.PlaybackPlaying, msg.at)
	if state.Playback.State != core.PlaybackPlaying && !m.settling() {
		return nil
	}
	return m.scheduleTick(m.tickInterval())
}

// scheduleTick invalidates any previously scheduled visualizer tick. Bubble
// Tea timers cannot be canceled, so generations make old messages inert.
func (m *Model) scheduleTick(after time.Duration) tea.Cmd {
	m.tickGeneration++
	generation := m.tickGeneration
	if after <= 0 {
		return func() tea.Msg {
			return tickMsg{at: time.Now(), generation: generation}
		}
	}
	return tea.Tick(after, func(t time.Time) tea.Msg {
		return tickMsg{at: t, generation: generation}
	})
}

func (m *Model) tickInterval() time.Duration {
	fps := min(max(m.fps, 1), 30)
	return time.Second / time.Duration(fps)
}

// advance accepts the newest polled snapshot and applies time-based smoothing.
// Resuming playback consumes the current sequence without displaying it; the
// analyzer must publish a later sequence before non-zero targets are accepted.
func (m *Model) advance(snapshot core.Spectrum, playing bool, now time.Time) {
	dt := now.Sub(m.lastUpdate)
	if m.lastUpdate.IsZero() || dt < 0 {
		dt = 0
	}
	m.lastUpdate = now

	valid := snapshot.PlaybackID != 0 && snapshot.Generation != 0
	identityChanged := valid && (snapshot.PlaybackID != m.playbackID || snapshot.Generation != m.generation)
	resumed := playing && !m.playing && !identityChanged && m.playbackID != 0
	m.playing = playing
	if identityChanged {
		clear(m.displayed[:])
		clear(m.targets[:])
		m.playbackID = snapshot.PlaybackID
		m.generation = snapshot.Generation
		m.sequence = 0
	}

	switch {
	case !playing || !valid:
		clear(m.targets[:])
		if valid {
			m.sequence = max(m.sequence, snapshot.Sequence)
		}
	case resumed:
		clear(m.targets[:])
		m.sequence = max(m.sequence, snapshot.Sequence)
	case snapshot.Sequence > m.sequence:
		for i, value := range snapshot.Bands {
			m.targets[i] = min(max(value, 0), 1)
		}
		m.sequence = snapshot.Sequence
	}

	for i, target := range m.targets {
		duration := releaseDuration
		if target > m.displayed[i] {
			duration = attackDuration
		}
		if dt > 0 {
			alpha := 1 - math.Exp(-float64(dt)/float64(duration))
			m.displayed[i] += (target - m.displayed[i]) * alpha
		}
		if m.displayed[i] < settledLevel/100 && target == 0 {
			m.displayed[i] = 0
		}
	}
}

func (m *Model) settling() bool {
	for _, value := range m.displayed {
		if value >= settledLevel {
			return true
		}
	}
	return false
}

// View renders one or two left-aligned rows of bars. Wide views keep the analyzer
// compact instead of stretching the fixed number of frequency bands.
func (m *Model) View(rows int) string {
	if m.width <= 0 || rows <= 0 {
		return ""
	}
	rows = min(rows, 2)
	layout := m.barLayout()
	rendered := make([]string, rows)
	for row := range rows {
		var line strings.Builder
		for _, value := range layout.levels {
			units := int(math.Round(value * float64(rows*(len(barLevels)-1))))
			units -= (rows - row - 1) * (len(barLevels) - 1)
			units = min(max(units, 0), len(barLevels)-1)
			bar := strings.Repeat(string(barLevels[units]), layout.barWidth)
			if units > 0 {
				bar = m.styleForEnergy(value).Render(bar)
			}
			line.WriteString(bar)
		}
		line.WriteString(strings.Repeat(" ", layout.rightPadding))
		rendered[row] = line.String()
	}
	return strings.Join(rendered, "\n")
}

func (m *Model) styleForEnergy(value float64) lipgloss.Style {
	value = min(max(value, 0), 1)
	index := int(math.Round(value * float64(len(m.energyStyles)-1)))
	return m.energyStyles[index]
}

type layout struct {
	levels       []float64
	barWidth     int
	rightPadding int
}

func (m *Model) barLayout() layout {
	if m.width < len(m.displayed) {
		return layout{
			levels:   m.combinedLevels(m.width),
			barWidth: 1,
		}
	}

	return layout{
		levels:       m.displayed[:],
		barWidth:     1,
		rightPadding: m.width - len(m.displayed),
	}
}

// combinedLevels reduces the analysis bands by average power, rather than
// averaging their logarithmic display values.
func (m *Model) combinedLevels(count int) []float64 {
	levels := make([]float64, count)
	for column := range levels {
		first := column * len(m.displayed) / count
		last := (column + 1) * len(m.displayed) / count

		power := 0.0
		for _, value := range m.displayed[first:last] {
			db := displayFloorDB + value*-displayFloorDB
			power += math.Pow(10, db/10)
		}
		power /= float64(last - first)
		levels[column] = min(max((10*math.Log10(power)-displayFloorDB)/-displayFloorDB, 0), 1)
	}
	return levels
}
