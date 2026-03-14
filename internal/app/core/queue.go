package core

// QueueMode identifies the playback ordering strategy.
type QueueMode int

const (
	QueueModeLinear QueueMode = iota
	QueueModeShuffle
	QueueModeRepeatOne
	QueueModeRepeatAll
	QueueModeStopAfterCurrent
)

// QueueInput is the immutable input provided to queue strategies.
type QueueInput struct {
	PlaylistLen int
	Playing     int
}

// QueueDecision describes the outcome of a queue step.
type QueueDecision struct {
	// Index is the track index to play next. If negative, no track should be played.
	Index int
	// Stop indicates playback should clear the current playing index when Index is negative.
	Stop bool
}

// QueuePlay returns a decision to play the provided index.
func QueuePlay(index int) QueueDecision {
	return QueueDecision{Index: index}
}

// QueueNoop returns a decision indicating no next/previous track.
func QueueNoop() QueueDecision {
	return QueueDecision{Index: -1}
}

// QueueStop returns a decision indicating playback should stop.
func QueueStop() QueueDecision {
	return QueueDecision{Index: -1, Stop: true}
}

// QueueStrategy selects next/previous indices based on state.
type QueueStrategy interface {
	// Next returns the next index to play, based on the intention to advance forward.
	Next(in QueueInput) QueueDecision
	// Prev returns the next index to play, based on the intention to advance backward.
	Prev(in QueueInput) QueueDecision
	// Reset resets the strategy's internal state, if any.
	Reset()
}
