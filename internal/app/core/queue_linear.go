package core

// LinearStrategy advances through the playlist in order.
type LinearStrategy struct{}

func (s LinearStrategy) Next(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if in.Playing == -1 {
		return QueuePlay(0)
	}
	if in.Playing+1 >= in.PlaylistLen {
		return QueueStop()
	}
	return QueuePlay(in.Playing + 1)
}

func (s LinearStrategy) Prev(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if in.Playing == -1 {
		return QueuePlay(0)
	}
	if in.Playing-1 < 0 {
		return QueueNoop()
	}
	return QueuePlay(in.Playing - 1)
}

func (s LinearStrategy) Reset() {
}
