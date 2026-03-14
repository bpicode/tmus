package core

// RepeatAllStrategy loops through the playlist in order.
type RepeatAllStrategy struct{}

func (s RepeatAllStrategy) Next(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if in.Playing == -1 {
		return QueuePlay(0)
	}
	return QueuePlay((in.Playing + 1) % in.PlaylistLen)
}

func (s RepeatAllStrategy) Prev(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if in.Playing == -1 {
		return QueuePlay(0)
	}
	prev := in.Playing - 1
	if prev < 0 {
		prev = in.PlaylistLen - 1
	}
	return QueuePlay(prev)
}

func (s RepeatAllStrategy) Reset() {
}
