package core

type StopAfterCurrentStrategy struct {
	linear LinearStrategy
}

func (s StopAfterCurrentStrategy) Next(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 || in.Playing < 0 {
		return QueueNoop()
	}
	return QueueStop()
}

func (s StopAfterCurrentStrategy) Prev(in QueueInput) QueueDecision {
	return s.linear.Prev(in)
}

func (s StopAfterCurrentStrategy) Reset() {
}
