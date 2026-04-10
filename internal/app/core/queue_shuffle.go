package core

import "math/rand/v2"

// maxShuffleHistory is the maximum number of entries retained in the
// shuffle back-navigation history. Older entries are discarded to prevent
// unbounded memory growth during long listening sessions.
const maxShuffleHistory = 500

// ShuffleStrategy advances through a shuffled deck of all tracks before
// reshuffling, so every track is heard exactly once per cycle.
// History is tracked independently for backward navigation via Prev.
type ShuffleStrategy struct {
	// deck is the shuffled permutation of track indices for the current cycle.
	deck []int
	// pos is the index of the next track to play within deck.
	pos int

	history [maxShuffleHistory]int
	head    int
	tail    int
	count   int
}

// NewShuffleStrategy constructs a new ShuffleStrategy.
func NewShuffleStrategy() *ShuffleStrategy {
	return &ShuffleStrategy{}
}

func (s *ShuffleStrategy) Next(in QueueInput) QueueDecision {
	total := in.PlaylistLen
	if total == 0 {
		return QueueNoop()
	}

	// Record the current track in history before advancing.
	if in.Playing != -1 {
		s.pushHistory(in.Playing)
	}

	// Rebuild the deck when the playlist size changed or the cycle is exhausted.
	if len(s.deck) != total || s.pos >= len(s.deck) {
		s.buildDeck(total, in.Playing)
	}

	next := s.deck[s.pos]
	s.pos++
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
	s.deck = s.deck[:0]
	s.pos = 0
	s.head = 0
	s.tail = 0
	s.count = 0
}

// buildDeck creates a freshly shuffled permutation of all track indices.
// The currently playing track is moved to the end so a new cycle never
// opens with the same song that just finished.
func (s *ShuffleStrategy) buildDeck(total, current int) {
	if cap(s.deck) >= total {
		s.deck = s.deck[:total]
	} else {
		s.deck = make([]int, total)
	}
	for i := range s.deck {
		s.deck[i] = i
	}
	rand.Shuffle(total, func(i, j int) {
		s.deck[i], s.deck[j] = s.deck[j], s.deck[i]
	})
	// Ensure the current track doesn't appear first in the new cycle.
	if current >= 0 {
		for i, v := range s.deck {
			if v == current {
				s.deck[i], s.deck[total-1] = s.deck[total-1], s.deck[i]
				break
			}
		}
	}
	s.pos = 0
}

func (s *ShuffleStrategy) pushHistory(index int) {
	s.history[s.tail] = index
	s.tail = (s.tail + 1) % maxShuffleHistory
	if s.count < maxShuffleHistory {
		s.count++
	} else {
		s.head = (s.head + 1) % maxShuffleHistory
	}
}
