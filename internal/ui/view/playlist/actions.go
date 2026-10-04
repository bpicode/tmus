package playlist

import tea "charm.land/bubbletea/v2"

type ToggleLyricsMsg struct{}

func toggleLyricsCmd() tea.Cmd {
	return func() tea.Msg {
		return ToggleLyricsMsg{}
	}
}

type ToggleTrackInfoMsg struct{}

func toggleTrackInfoCmd() tea.Cmd {
	return func() tea.Msg {
		return ToggleTrackInfoMsg{}
	}
}
