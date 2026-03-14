package core

// RepeatOneStrategy keeps playback on the current track.
type RepeatOneStrategy struct{}

func (s RepeatOneStrategy) Next(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if in.Playing == -1 {
		return QueuePlay(0)
	}
	return QueuePlay(in.Playing)
}

func (s RepeatOneStrategy) Prev(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if in.Playing == -1 {
		return QueuePlay(0)
	}
	return QueuePlay(in.Playing)
}

func (s RepeatOneStrategy) Reset() {
}
