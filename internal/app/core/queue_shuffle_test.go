package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
