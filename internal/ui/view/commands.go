package view

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/ui/components/notification"
)

func (m *Model) showTrackInfo() tea.Cmd {
	cmd, request := m.trackInfo.Show(m.state)
	if request == nil {
		return cmd
	}
	err := m.app.Dispatch(core.Command{
		Type:    core.CmdRequestMetadata,
		TrackID: request.TrackID,
		Path:    request.Path,
		Scope:   core.MetadataExtended,
	})
	if err == nil {
		return cmd
	}
	err = fmt.Errorf("request track metadata: %w", err)
	_, failedCmd, _ := m.trackInfo.Update(core.MetadataEvent{
		TrackID: request.TrackID,
		Path:    request.Path,
		Scope:   core.MetadataExtended,
		Err:     err,
	})
	return tea.Batch(cmd, failedCmd, notification.Error(err))
}

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
