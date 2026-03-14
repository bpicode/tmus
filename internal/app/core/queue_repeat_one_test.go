package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepeatOneStrategy_Next(t *testing.T) {
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
			name: "stay at currently playing track",
			arg:  QueueInput{PlaylistLen: 15, Playing: 7},
			want: QueuePlay(7),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RepeatOneStrategy{}
			n := s.Next(tt.arg)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestRepeatOneStrategy_Prev(t *testing.T) {
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
			name: "stay at currently playing track",
			arg:  QueueInput{PlaylistLen: 15, Playing: 7},
			want: QueuePlay(7),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RepeatOneStrategy{}
			p := s.Prev(tt.arg)
			assert.Equal(t, tt.want, p)
		})
	}
}
