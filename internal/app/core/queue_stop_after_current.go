package core

type StopAfterCurrentStrategy struct {
	linear LinearStrategy
}

func (s StopAfterCurrentStrategy) Next(_ QueueInput) QueueDecision {
	return QueueNoop()
}

func (s StopAfterCurrentStrategy) Prev(in QueueInput) QueueDecision {
	return s.linear.Prev(in)
}

func (s StopAfterCurrentStrategy) Reset() {
}
