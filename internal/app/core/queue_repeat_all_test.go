package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepeatAllStrategy_Next(t *testing.T) {
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
			arg:  QueueInput{PlaylistLen: 3, Playing: -1},
			want: QueuePlay(0),
		},
		{
			name: "advance to next track",
			arg:  QueueInput{PlaylistLen: 15, Playing: 1},
			want: QueuePlay(2),
		},
		{
			name: "start from top after last track",
			arg:  QueueInput{PlaylistLen: 20, Playing: 19},
			want: QueuePlay(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RepeatAllStrategy{}
			n := s.Next(tt.arg)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestRepeatAllStrategy_Prev(t *testing.T) {
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
			arg:  QueueInput{PlaylistLen: 3, Playing: -1},
			want: QueuePlay(0),
		},
		{
			name: "move to previous track",
			arg:  QueueInput{PlaylistLen: 15, Playing: 6},
			want: QueuePlay(5),
		},
		{
			name: "move to bottom after first track",
			arg:  QueueInput{PlaylistLen: 20, Playing: 0},
			want: QueuePlay(19),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RepeatAllStrategy{}
			p := s.Prev(tt.arg)
			assert.Equal(t, tt.want, p)
		})
	}
}
