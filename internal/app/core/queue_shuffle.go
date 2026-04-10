package core

import "math/rand/v2"

// maxShuffleHistory is the maximum number of entries retained in the
// shuffle back-navigation history. Older entries are discarded to prevent
// unbounded memory growth during long listening sessions.
const maxShuffleHistory = 500

// ShuffleStrategy advances randomly and tracks history for prev.
type ShuffleStrategy struct {
	history [maxShuffleHistory]int
	head    int
	tail    int
	count   int
}

// NewShuffleStrategy constructs a shuffle strategy with a random seed.
func NewShuffleStrategy() *ShuffleStrategy {
	return &ShuffleStrategy{}
}

func (s *ShuffleStrategy) Next(in QueueInput) QueueDecision {
	total := in.PlaylistLen
	if total == 0 {
		return QueueNoop()
	}
	if in.Playing != -1 {
		s.history[s.tail] = in.Playing
		s.tail = (s.tail + 1) % maxShuffleHistory
		if s.count < maxShuffleHistory {
			s.count++
		} else {
			s.head = (s.head + 1) % maxShuffleHistory
		}
	}
	next := s.pickNext(total, in.Playing)
	return QueuePlay(next)
}

func (s *ShuffleStrategy) Prev(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if s.count == 0 {
		if in.Playing == -1 {
			return QueueNoop()
		}
		return QueuePlay(in.Playing)
	}
	lastIdx := (s.tail - 1 + maxShuffleHistory) % maxShuffleHistory
	last := s.history[lastIdx]
	s.tail = lastIdx
	s.count--

	if last < 0 || last >= in.PlaylistLen {
		return QueueStop()
	}
	return QueuePlay(last)
}

func (s *ShuffleStrategy) Reset() {
	s.head = 0
	s.tail = 0
	s.count = 0
}

func (s *ShuffleStrategy) pickNext(total, current int) int {
	if total <= 1 {
		return 0
	}
	if current == -1 {
		return rand.IntN(total)
	}
	// Pick uniformly from [0, total-1) then map to [0, total) excluding current,
	// so the result is always different from current and always O(1).
	n := rand.IntN(total - 1)
	if n >= current {
		n++
	}
	return n
}
