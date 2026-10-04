package view

import tea "charm.land/bubbletea/v2"

// Child command replies retain their origin even after switching tabs.
type browserMsg struct{ msg tea.Msg }
type playlistMsg struct{ msg tea.Msg }

func browserCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		return browserMsg{msg: cmd()}
	}
}

func playlistCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		return playlistMsg{msg: cmd()}
	}
}

// Let Bubble Tea execute a child's batch, preserving the origin of each reply.
func childBatch(batch tea.BatchMsg, wrap func(tea.Cmd) tea.Cmd) tea.Cmd {
	cmds := make([]tea.Cmd, len(batch))
	for i, cmd := range batch {
		cmds[i] = wrap(cmd)
	}
	return tea.Batch(cmds...)
}
