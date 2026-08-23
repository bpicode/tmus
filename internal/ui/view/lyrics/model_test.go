package lyrics

import (
	"testing"
	"time"

	applyrics "github.com/bpicode/tmus/internal/app/lyrics"
	"github.com/stretchr/testify/assert"
)

func TestNextLyricDelay(t *testing.T) {
	lines := []applyrics.Line{
		{Text: "untimed"},
		{Text: "third", Time: 3 * time.Second, HasTime: true},
		{Text: "first", Time: time.Second, HasTime: true},
		{Text: "second", Time: 2250 * time.Millisecond, HasTime: true},
		{Text: "second duplicate", Time: 2250 * time.Millisecond, HasTime: true},
	}

	tests := []struct {
		name    string
		elapsed time.Duration
		want    time.Duration
		wantOK  bool
	}{
		{name: "before first", elapsed: 250 * time.Millisecond, want: 750 * time.Millisecond, wantOK: true},
		{name: "within one frame", elapsed: 995 * time.Millisecond, want: minLineTickDelay, wantOK: true},
		{name: "at first", elapsed: time.Second, want: 1250 * time.Millisecond, wantOK: true},
		{name: "between lines", elapsed: 2 * time.Second, want: 250 * time.Millisecond, wantOK: true},
		{name: "at duplicate lines", elapsed: 2250 * time.Millisecond, want: 750 * time.Millisecond, wantOK: true},
		{name: "after last", elapsed: 4 * time.Second, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := nextLyricDelay(lines, tt.elapsed)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNextLyricDelayWithNoTimedLines(t *testing.T) {
	lines := []applyrics.Line{{Text: "first"}, {Text: "second"}}

	delay, ok := nextLyricDelay(lines, 0)

	assert.False(t, ok)
	assert.Zero(t, delay)
}

func TestActiveLyricIndex(t *testing.T) {
	lines := []applyrics.Line{
		{Text: "untimed"},
		{Text: "third", Time: 3 * time.Second, HasTime: true},
		{Text: "first", Time: time.Second, HasTime: true},
		{Text: "second", Time: 2250 * time.Millisecond, HasTime: true},
		{Text: "second duplicate", Time: 2250 * time.Millisecond, HasTime: true},
	}

	tests := []struct {
		name    string
		elapsed time.Duration
		want    int
	}{
		{name: "before first", elapsed: 500 * time.Millisecond, want: -1},
		{name: "at first", elapsed: time.Second, want: 2},
		{name: "between first and second", elapsed: 2 * time.Second, want: 2},
		{name: "duplicate timestamp uses later line", elapsed: 2500 * time.Millisecond, want: 4},
		{name: "unsorted later timestamp", elapsed: 3 * time.Second, want: 1},
		{name: "after last", elapsed: 4 * time.Second, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, activeLyricIndex(lines, tt.elapsed))
		})
	}
}

func TestModelIgnoresStaleLineTick(t *testing.T) {
	m := &Model{lineTickGeneration: 2}

	got, cmd, stop := m.Update(lineTickMsg{generation: 1})

	assert.Same(t, m, got)
	assert.Nil(t, cmd)
	assert.False(t, stop)
	assert.Equal(t, uint64(2), m.lineTickGeneration)
}

func TestLineTickCmdReturnsMessage(t *testing.T) {
	cancel := make(chan struct{})

	msg := lineTickCmd(0, 7, cancel)()

	assert.Equal(t, lineTickMsg{generation: 7}, msg)
}

func TestLineTickCmdCanBeCanceled(t *testing.T) {
	cancel := make(chan struct{})
	close(cancel)

	msg := lineTickCmd(time.Hour, 7, cancel)()

	assert.Nil(t, msg)
}

func TestInvalidateLineTickCancelsPendingCommand(t *testing.T) {
	cancel := make(chan struct{})
	m := &Model{lineTickGeneration: 2, lineTickCancel: cancel}

	m.invalidateLineTick()

	assert.Equal(t, uint64(3), m.lineTickGeneration)
	assert.Nil(t, m.lineTickCancel)
	assert.True(t, channelClosed(cancel))
}

func channelClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
