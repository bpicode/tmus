package playlist

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	titleUnfocused  lipgloss.Style
	titleFocused    lipgloss.Style
	searchInactive  lipgloss.Style
	searchActive    lipgloss.Style
	err             lipgloss.Style
	empty           lipgloss.Style
	selected        lipgloss.Style
	playing         lipgloss.Style
	paused          lipgloss.Style
	statusQueueMode lipgloss.Style
	statusPlay      lipgloss.Style
	statusPause     lipgloss.Style
	statusStop      lipgloss.Style
	separator       lipgloss.Style
	panelFocused    lipgloss.Style
	panelUnfocused  lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		titleUnfocused:  lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		titleFocused:    lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		searchInactive:  lipgloss.NewStyle().Foreground(th.Muted),
		searchActive:    lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		err:             lipgloss.NewStyle().Foreground(th.Danger),
		empty:           lipgloss.NewStyle().Foreground(th.Muted),
		selected:        lipgloss.NewStyle().Reverse(true),
		playing:         lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		paused:          lipgloss.NewStyle().Bold(true).Foreground(th.Warning),
		statusQueueMode: lipgloss.NewStyle().Foreground(th.Highlight),
		statusPlay:      lipgloss.NewStyle().Foreground(th.Info),
		statusPause:     lipgloss.NewStyle().Foreground(th.Warning),
		statusStop:      lipgloss.NewStyle().Foreground(th.Danger),
		separator:       lipgloss.NewStyle().Foreground(th.Muted),
		panelFocused:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary).Padding(0, 1),
		panelUnfocused:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Muted).Padding(0, 1),
	}
}

func (s styles) playStateStyle(state core.State) lipgloss.Style {
	switch state.Playback.State {
	case core.PlaybackPaused:
		return s.statusPause
	case core.PlaybackPlaying:
		return s.statusPlay
	default:
		return s.statusStop
	}
}
