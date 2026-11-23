package player

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/gopxl/beep/v2"
	"github.com/stretchr/testify/assert"
)

func TestEngineCloseAndWaitFinishesCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := NewEngine(Options{SampleRate: 44100})
		streamer := &shutdownStreamer{entered: make(chan struct{}), release: make(chan struct{})}
		e.mu.Lock()
		e.streamer = streamer
		e.mu.Unlock()
		e.Close()
		<-streamer.entered

		finished := make(chan struct{})
		go func() {
			e.CloseAndWait()
			close(finished)
		}()
		synctest.Wait()
		select {
		case <-finished:
			t.Error("CloseAndWait returned before the stream was closed")
		default:
		}
		close(streamer.release)
		<-finished
		_, open := <-e.Events()
		assert.False(t, open, "engine events are closed when cleanup completes")
		assert.True(t, e.analyzer.closed.Load(), "the analyzer is stopped")
	})
}

func TestEngineCloseAndWaitConcurrentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := NewEngine(Options{SampleRate: 44100})
		var wg sync.WaitGroup
		for range 3 {
			wg.Go(e.CloseAndWait)
		}
		wg.Wait()
		_, open := <-e.Events()
		assert.False(t, open)
	})
}

func TestEngineSeekReturnsOnShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		e := &Engine{ctx: ctx, cancel: cancel, cmdCh: make(chan command, 1)}
		result := make(chan SeekResult, 1)
		go func() { result <- e.SeekTo(0) }()
		<-e.cmdCh // Accept the seek without sending a reply.
		e.Close()
		assert.Equal(t, SeekResult{}, <-result)
	})
}

type shutdownStreamer struct {
	beep.StreamSeekCloser
	entered chan struct{}
	release chan struct{}
}

func (s *shutdownStreamer) Close() error {
	close(s.entered)
	<-s.release
	return nil
}
