package view

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
)

type eventChannels struct {
	state         <-chan core.StateEvent
	unsubState    func()
	metadata      <-chan core.MetadataEvent
	unsubMetadata func()
	lyrics        <-chan core.LyricsEvent
	unsubLyrics   func()
}

// Closed subscriptions produce a distinct message rather than a zero-valued event.
type stateClosedMsg struct{}
type metadataClosedMsg struct{}
type lyricsClosedMsg struct{}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
