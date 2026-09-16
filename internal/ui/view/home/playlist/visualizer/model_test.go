package visualizer

import (
	"image/color"
	"math"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/player"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAttackReleaseAndReset(t *testing.T) {
	m := New(Config{})
	now := time.Unix(100, 0)
	snapshot := core.Spectrum{PlaybackID: 1, Generation: 1, Sequence: 1}
	snapshot.Bands[0] = 1

	m.advance(snapshot, true, now)
	m.advance(snapshot, true, now.Add(attackDuration))
	wantAttack := 1 - math.Exp(-1)
	assert.InDelta(t, wantAttack, m.displayed[0], 1e-9)

	m.advance(snapshot, false, now.Add(attackDuration+releaseDuration))
	wantRelease := wantAttack * math.Exp(-1)
	assert.InDelta(t, wantRelease, m.displayed[0], 1e-9)

	reset := core.Spectrum{PlaybackID: 1, Generation: 2, Sequence: 2}
	m.advance(reset, true, now.Add(attackDuration+releaseDuration+time.Millisecond))
	assert.Zero(t, m.displayed)
}

func TestPausedSnapshotRequiresFreshSequence(t *testing.T) {
	m := New(Config{})
	now := time.Unix(100, 0)
	snapshot := core.Spectrum{PlaybackID: 1, Generation: 1, Sequence: 8}
	snapshot.Bands[0] = 1

	m.advance(snapshot, true, now)
	m.advance(snapshot, true, now.Add(attackDuration))
	snapshot.Sequence++
	m.advance(snapshot, false, now.Add(2*attackDuration))
	snapshot.Sequence++ // May have completed from samples captured before pause.
	m.advance(snapshot, true, now.Add(3*attackDuration))
	assert.Zero(t, m.targets[0])

	snapshot.Sequence++
	m.advance(snapshot, true, now.Add(4*attackDuration))
	assert.Equal(t, 1.0, m.targets[0])

	stale := snapshot
	stale.Sequence--
	stale.Bands[0] = 0.25
	m.advance(stale, true, now.Add(5*attackDuration))
	assert.Equal(t, 1.0, m.targets[0])
}

func TestViewUsesStackedEighthBlocks(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "empty", want: " \n "},
		{name: "bottom row", value: 0.5, want: " \n█"},
		{name: "crosses into top row", value: 9.0 / 16, want: "▁\n█"},
		{name: "full", value: 1, want: "█\n█"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(Config{})
			for i := range m.displayed {
				m.displayed[i] = tt.value
			}
			assert.Equal(t, tt.want, m.View(1, 2))
		})
	}
}

func TestViewWidthAndNarrowBandCombination(t *testing.T) {
	m := New(Config{})
	m.displayed[0] = 1

	combined := m.combinedLevels(1)
	power := (1 + float64(len(m.displayed)-1)*math.Pow(10, displayFloorDB/10)) / float64(len(m.displayed))
	wantCombined := (10*math.Log10(power) - displayFloorDB) / -displayFloorDB
	assert.InDelta(t, wantCombined, combined[0], 1e-12)

	for _, width := range []int{1, 8, len(m.displayed), len(m.displayed) + 15, 100} {
		view := m.View(width, 2)
		for row, line := range strings.Split(view, "\n") {
			assert.Equalf(t, width, lipgloss.Width(line), "row %d", row)
		}
	}
}

func TestBarLayoutKeepsWideViewsCompact(t *testing.T) {
	bandCount := core.SpectrumBandCount
	tests := []struct {
		name            string
		width           int
		bands, barWidth int
		rightPadding    int
	}{
		{name: "combine narrow bands", width: bandCount - 1, bands: bandCount - 1, barWidth: 1},
		{name: "adjacent bars", width: bandCount, bands: bandCount, barWidth: 1},
		{name: "cap full-width view", width: 100, bands: bandCount, barWidth: 1, rightPadding: 100 - bandCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(Config{}).barLayout(tt.width)
			assert.Len(t, got.levels, tt.bands)
			assert.Equal(t, tt.barWidth, got.barWidth)
			assert.Equal(t, tt.rightPadding, got.rightPadding)
		})
	}
}

func TestViewPlacesFrequencyBandsTogetherAtLeft(t *testing.T) {
	m := New(Config{})
	for i := range m.displayed {
		m.displayed[i] = 1
	}

	width := len(m.displayed) + 15
	want := strings.Repeat("█", len(m.displayed)) + strings.Repeat(" ", 15)
	assert.Equal(t, want, m.View(width, 1))
}

func TestEnergyStylesBlendThemeAccents(t *testing.T) {
	primary := lipgloss.Color("#204060")
	secondary := lipgloss.Color("#80a0c0")
	m := New(Config{Theme: theme.Theme{Primary: primary, Secondary: secondary}})

	low := color.RGBAModel.Convert(m.styleForEnergy(0).GetForeground())
	middle := color.RGBAModel.Convert(m.styleForEnergy(0.5).GetForeground())
	high := color.RGBAModel.Convert(m.styleForEnergy(1).GetForeground())

	assert.Equal(t, color.RGBAModel.Convert(primary), low)
	assert.NotEqual(t, low, middle)
	assert.NotEqual(t, high, middle)
	assert.Equal(t, color.RGBAModel.Convert(secondary), high)
	assert.Equal(t, low, color.RGBAModel.Convert(m.styleForEnergy(-1).GetForeground()))
	assert.Equal(t, high, color.RGBAModel.Convert(m.styleForEnergy(2).GetForeground()))
}

func TestSettlingThreshold(t *testing.T) {
	m := New(Config{})
	assert.False(t, m.settling())
	m.displayed[0] = settledLevel
	assert.True(t, m.settling())
}

func TestAdvanceClampsSnapshotLevels(t *testing.T) {
	m := New(Config{})
	snapshot := core.Spectrum{PlaybackID: 1, Generation: 1, Sequence: 1}
	snapshot.Bands[0] = -1
	snapshot.Bands[1] = 2

	m.advance(snapshot, true, time.Unix(100, 0))

	assert.Zero(t, m.targets[0])
	assert.Equal(t, 1.0, m.targets[1])
}

func TestTickInterval(t *testing.T) {
	tests := []struct {
		name string
		fps  int
		want time.Duration
	}{
		{name: "capped at thirty", fps: 60, want: time.Second / 30},
		{name: "uses lower configured fps", fps: 12, want: time.Second / 12},
		{name: "clamps invalid low fps", fps: 0, want: time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(Config{FPS: tt.fps})
			assert.Equal(t, tt.want, m.tickInterval())
		})
	}
}

func TestScheduleTickInvalidatesEarlierTick(t *testing.T) {
	m := New(Config{})
	firstMsg := m.scheduleTick(0)()
	secondMsg := m.scheduleTick(0)()
	require.IsType(t, tickMsg{}, firstMsg)
	require.IsType(t, tickMsg{}, secondMsg)
	first := firstMsg.(tickMsg)
	second := secondMsg.(tickMsg)

	assert.NotEqual(t, first.generation, second.generation)
	assert.Equal(t, m.tickGeneration, second.generation)
}

func TestUpdateIgnoresStaleTick(t *testing.T) {
	m := New(Config{})
	m.tickGeneration = 2

	_, cmd := m.Update(tickMsg{at: time.Now(), generation: 1})

	assert.Nil(t, cmd)
}

func TestTickStopsWhenPlaybackIsIdle(t *testing.T) {
	m, _ := newTickTestModel(t, 60)
	got := m.scheduleTick(0)()
	require.IsType(t, tickMsg{}, got)
	msg := got.(tickMsg)

	_, cmd := m.Update(msg)

	assert.Nil(t, cmd)
}

func TestTickContinuesWhilePlaying(t *testing.T) {
	m, app := newTickTestModel(t, 60)
	app.HandlePlayerEvent(player.Event{Type: player.EventTrackStarted, PlaybackID: 1})
	got := m.scheduleTick(0)()
	require.IsType(t, tickMsg{}, got)
	msg := got.(tickMsg)

	_, cmd := m.Update(msg)

	assert.NotNil(t, cmd)
}

func TestTickContinuesWhileSettling(t *testing.T) {
	m, _ := newTickTestModel(t, 60)
	got := m.scheduleTick(0)()
	require.IsType(t, tickMsg{}, got)
	msg := got.(tickMsg)
	m.displayed[0] = 1
	m.lastUpdate = msg.at.Add(-releaseDuration)

	_, cmd := m.Update(msg)

	assert.Less(t, m.displayed[0], 1.0)
	assert.NotNil(t, cmd)
}

func TestPlaybackEventRestartsTick(t *testing.T) {
	m := New(Config{})

	_, cmd := m.Update(core.StateEvent{Changes: core.StateChangePlaying})

	require.NotNil(t, cmd)
	assert.IsType(t, tickMsg{}, cmd())
}

func newTickTestModel(t *testing.T, fps int) (*Model, *core.App) {
	t.Helper()
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Lyrics.LrcLib.Enabled = false
	app := core.New(cfg)
	t.Cleanup(app.ShutdownAndWait)
	return New(Config{App: app, FPS: fps}), app
}
