package core

import (
	"math/rand"
	"time"
)

// maxShuffleHistory is the maximum number of entries retained in the
// shuffle back-navigation history. Older entries are discarded to prevent
// unbounded memory growth during long listening sessions.
const maxShuffleHistory = 500

// ShuffleStrategy advances randomly and tracks history for prev.
type ShuffleStrategy struct {
	history []int
	rng     *rand.Rand
}

// NewShuffleStrategy constructs a shuffle strategy with a random seed.
func NewShuffleStrategy() *ShuffleStrategy {
	return &ShuffleStrategy{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *ShuffleStrategy) Next(in QueueInput) QueueDecision {
	total := in.PlaylistLen
	if total == 0 {
		return QueueNoop()
	}
	if in.Playing != -1 {
		s.history = append(s.history, in.Playing)
		if len(s.history) > maxShuffleHistory {
			s.history = s.history[len(s.history)-maxShuffleHistory:]
		}
	}
	next := s.pickNext(total, in.Playing)
	return QueuePlay(next)
}

func (s *ShuffleStrategy) Prev(in QueueInput) QueueDecision {
	if in.PlaylistLen == 0 {
		return QueueNoop()
	}
	if len(s.history) == 0 {
		if in.Playing == -1 {
			return QueueNoop()
		}
		return QueuePlay(in.Playing)
	}
	last := s.history[len(s.history)-1]
	s.history = s.history[:len(s.history)-1]
	if last < 0 || last >= in.PlaylistLen {
		return QueueStop()
	}
	return QueuePlay(last)
}

func (s *ShuffleStrategy) Reset() {
	s.history = nil
}

func (s *ShuffleStrategy) pickNext(total, current int) int {
	if total <= 1 {
		return 0
	}
	// Pick uniformly from [0, total-1) then map to [0, total) excluding current,
	// so the result is always different from current and always O(1).
	n := s.rng.Intn(total - 1)
	if n >= current {
		n++
	}
	return n
}
