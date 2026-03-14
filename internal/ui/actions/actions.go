package actions

import tea "charm.land/bubbletea/v2"

type ToggleLyricsMsg struct{}

func ToggleLyricsCmd() tea.Cmd {
	return func() tea.Msg {
		return ToggleLyricsMsg{}
	}
}

type ToggleTrackInfoMsg struct{}

func ToggleTrackInfoCmd() tea.Cmd {
	return func() tea.Msg {
		return ToggleTrackInfoMsg{}
	}
}
