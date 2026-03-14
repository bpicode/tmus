package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinearStrategy_Next(t *testing.T) {
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
			name: "stop after last track",
			arg:  QueueInput{PlaylistLen: 20, Playing: 19},
			want: QueueStop(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LinearStrategy{}
			n := s.Next(tt.arg)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestLinearStrategy_Prev(t *testing.T) {
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
			name: "noop before first track",
			arg:  QueueInput{PlaylistLen: 20, Playing: 0},
			want: QueueNoop(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LinearStrategy{}
			p := s.Prev(tt.arg)
			assert.Equal(t, tt.want, p)
		})
	}
}
