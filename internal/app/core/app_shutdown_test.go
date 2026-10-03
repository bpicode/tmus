package core

import (
	"sync"
	"testing"
	"testing/synctest"

	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestAppShutdownAndWaitFinishesForwarding(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := New(config.Default())
		state, _ := app.SubscribeStateEvents()
		player, _ := app.SubscribePlayerEvents()
		metadata, _ := app.SubscribeMetadataEvents()
		lyrics, _ := app.SubscribeLyricsEvents()
		app.Restore([]Track{{Path: "/music/one.flac", Name: "One"}}, 0, QueueModeLinear)
		track := app.State().Playlist[0]
		app.metadataChan <- MetadataEvent{
			TrackID: track.ID, Path: track.Path,
			Metadata: library.Metadata{Title: "Buffered title"},
		}
		app.ShutdownAndWait()
		assert.Equal(t, "Buffered title", app.State().Playlist[0].Title)
		assertStreamClosed(t, "state", state)
		assertStreamClosed(t, "player", player)
		assertStreamClosed(t, "metadata", metadata)
		assertStreamClosed(t, "lyrics", lyrics)
		assert.ErrorIs(t, app.Dispatch(Command{Type: CmdClear}), ErrAppClosed)
	})
}

func TestAppShutdownAndWaitConcurrentCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := New(config.Default())
		events, _ := app.SubscribeStateEvents()
		var wg sync.WaitGroup
		for range 3 {
			wg.Go(app.ShutdownAndWait)
		}
		wg.Wait()
		assertStreamClosed(t, "state", events)
	})
}

func assertStreamClosed[T any](t *testing.T, name string, events <-chan T) {
	t.Helper()
	for {
		select {
		case _, open := <-events:
			if !open {
				return
			}
		default:
			t.Errorf("%s events are still open after shutdown completed", name)
			return
		}
	}
}
