package status

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	statusPlay  lipgloss.Style
	statusPause lipgloss.Style
	statusStop  lipgloss.Style
	statusNone  lipgloss.Style
	statusTime  lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		statusPlay:  lipgloss.NewStyle().Foreground(th.Info),
		statusPause: lipgloss.NewStyle().Foreground(th.Warning),
		statusStop:  lipgloss.NewStyle().Foreground(th.Danger),
		statusNone:  lipgloss.NewStyle().Foreground(th.Muted),
		statusTime:  lipgloss.NewStyle().Foreground(th.Working),
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
