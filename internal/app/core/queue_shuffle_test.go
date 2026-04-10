package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShuffleStrategy_Next(t *testing.T) {
	tests := []struct {
		name string
		arg  QueueInput
		want QueueDecision
	}{
		{
			name: "empty playlist",
			arg:  QueueInput{PlaylistLen: 0, Playing: -1},
			want: QueueNoop(),
		},
		{
			name: "start playing",
			arg:  QueueInput{PlaylistLen: 1, Playing: -1},
			want: QueuePlay(0),
		},
		{
			name: "advance to different track",
			arg:  QueueInput{PlaylistLen: 2, Playing: 1},
			want: QueuePlay(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewShuffleStrategy()
			n := s.Next(tt.arg)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestShuffleStrategy_Prev(t *testing.T) {
	tests := []struct {
		name string
		arg  QueueInput
		want QueueDecision
	}{
		{
			name: "empty playlist",
			arg:  QueueInput{PlaylistLen: 0, Playing: -1},
			want: QueueNoop(),
		},
		{
			name: "noop when no history for previous track",
			arg:  QueueInput{PlaylistLen: 72, Playing: -1},
			want: QueueNoop(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewShuffleStrategy()
			p := s.Prev(tt.arg)
			assert.Equal(t, tt.want, p)
		})
	}
}

func TestNewShuffleStrategy_Next_And_Prev(t *testing.T) {
	s := NewShuffleStrategy()
	qi := QueueInput{PlaylistLen: 2, Playing: 0}
	n := s.Next(qi)
	assert.Equal(t, QueuePlay(1), n)

	qi.Playing = 1
	p := s.Prev(qi)
	assert.Equal(t, QueuePlay(0), p)
}

// TestShuffleStrategy_FullCycle verifies every track appears exactly once
// before the deck reshuffles (fair shuffle guarantee).
func TestShuffleStrategy_FullCycle(t *testing.T) {
	const n = 10
	s := NewShuffleStrategy()
	seen := make(map[int]int)
	playing := -1
	for range n {
		d := s.Next(QueueInput{PlaylistLen: n, Playing: playing})
		require.True(t, d.Index >= 0 && d.Index < n, "index %d out of range", d.Index)
		seen[d.Index]++
		playing = d.Index
	}
	for i := range n {
		assert.Equal(t, 1, seen[i], "track %d should appear exactly once in a cycle", i)
	}
}

// TestShuffleStrategy_PrevDoesNotAffectDeck verifies that calling Prev does
// not corrupt forward navigation: after going back via history, Next should
// still produce a valid in-range track.
func TestShuffleStrategy_PrevDoesNotAffectDeck(t *testing.T) {
	const n = 5
	s := NewShuffleStrategy()
	playing := -1

	// Advance two steps.
	for range 2 {
		d := s.Next(QueueInput{PlaylistLen: n, Playing: playing})
		require.True(t, d.Index >= 0 && d.Index < n)
		playing = d.Index
	}
	// Go back once.
	p := s.Prev(QueueInput{PlaylistLen: n, Playing: playing})
	require.True(t, p.Index >= 0 && p.Index < n)
	playing = p.Index

	// Going forward again should still yield a valid track (deck untouched).
	d := s.Next(QueueInput{PlaylistLen: n, Playing: playing})
	assert.True(t, d.Index >= 0 && d.Index < n, "Next after Prev should return a valid track")
}
