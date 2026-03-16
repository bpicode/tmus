package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bpicode/tmus/internal/app/lyrics"
	"github.com/bpicode/tmus/internal/app/player"
	"github.com/bpicode/tmus/internal/config"
)

// PlaybackState describes the current playback mode.
type PlaybackState int

const (
	PlaybackStopped PlaybackState = iota
	PlaybackPlaying
	PlaybackPaused
)

// State is the shared application state for the UI.
type State struct {
	Playlist     []Track
	PlaylistErr  error
	Playing      int
	Cursor       int
	QueueMode    QueueMode
	PlayState    PlaybackState
	PlayTrack    string
	PlayStart    time.Time
	PlayDuration time.Duration
	PausedAt     time.Time
	PausedFor    time.Duration
	Volume       int
}

// Elapsed returns the playback time elapsed for the current track.
func (s *State) Elapsed() time.Duration {
	elapsed := time.Since(s.PlayStart) - s.PausedFor
	if s.PlayState == PlaybackPaused && !s.PausedAt.IsZero() {
		elapsed = s.PausedAt.Sub(s.PlayStart) - s.PausedFor
	}
	return elapsed
}

// CommandType identifies a UI command.
type CommandType int

const (
	CmdAdd CommandType = iota
	CmdAddAll
	CmdClear
	CmdSelectUp
	CmdSelectDown
	CmdSelectTop
	CmdSelectBottom
	CmdSelectPageUp
	CmdSelectPageDown
	CmdSelectIndex
	CmdPlayFromCursor
	CmdNext
	CmdPrev
	CmdStop
	CmdTogglePause
	CmdRemoveAt
	CmdMoveUp
	CmdMoveDown
	CmdSetQueueMode
	CmdVolumeUp
	CmdVolumeDown
	CmdToggleMute
	CmdSetVolume
	CmdSeekBy
	CmdRequestMetadata
	CmdRequestLyrics
)

// Command represents a UI action to apply.
type Command struct {
	Type    CommandType
	Track   Track
	Tracks  []Track
	Index   int
	Mode    QueueMode
	Count   int
	Offset  time.Duration
	Volume  int
	TrackID uint64
	Path    string
	Scope   MetadataScope
}

// StateChange identifies which parts of the app state changed.
type StateChange uint32

const (
	StateChangeNone     StateChange = 0
	StateChangePlaylist StateChange = 1 << iota
	StateChangeSelection
	StateChangePlaying
	StateChangePlayback
	StateChangeQueue
	StateChangeVolume
	StateChangeMetadata
	StateChangeError
)

// StateEventSource indicates the origin of a state change.
type StateEventSource int

const (
	StateEventCommand StateEventSource = iota
	StateEventPlayer
	StateEventMetadata
)

// StateEvent signals that the app state was updated after a command or event.
type StateEvent struct {
	Source  StateEventSource
	Command Command
	Changes StateChange
}

// App owns playback and playlist state.
type App struct {
	state   State
	stateMu sync.RWMutex

	engine *player.Engine
	queue  QueueStrategy

	lastVolume  int
	nextTrackID uint64

	cmdChan    chan Command
	commandWG  sync.WaitGroup
	subsMu     sync.Mutex
	playerSubs map[chan player.Event]struct{}
	stateSubs  map[chan StateEvent]struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWG sync.WaitGroup
	closed     atomic.Bool

	metadataChan  chan TrackMetadataEvent
	metadataQueue chan metadataRequest
	metadataCache *metadataCache
	metadataWG    sync.WaitGroup
	metadataSubs  map[chan TrackMetadataEvent]struct{}

	lyricsChan     chan LyricsEvent
	lyricsQueue    chan lyricsRequest
	lyricsResolver *lyrics.Resolver
	lyricsCache    *lyricsCache
	lyricsWG       sync.WaitGroup
	lyricsSubs     map[chan LyricsEvent]struct{}
}

var (
	// ErrAppClosed is returned when dispatching to a closed app.
	ErrAppClosed = errors.New("app is closed")
	// ErrCommandQueueFull is returned when the command queue is full.
	ErrCommandQueueFull = errors.New("command queue full")
)

// DefaultVolume is the initial volume level (0-100).
const DefaultVolume = 100

const (
	metadataWorkerCount = 1
	lyricsWorkerCount   = 1
)

type metadataRequest struct {
	TrackID uint64
	Path    string
	Scope   MetadataScope
}

// New constructs a new App.
func New(cfg config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		engine: player.NewEngine(player.Options{
			SampleRate:      cfg.Audio.SampleRate,
			ResampleQuality: cfg.Audio.ResampleQuality,
			BufferDuration:  time.Duration(cfg.Audio.BufferMs) * time.Millisecond,
		}),
		state: State{
			Playing:   -1,
			Cursor:    -1,
			QueueMode: QueueModeLinear,
			PlayState: PlaybackStopped,
			Volume:    DefaultVolume,
		},
		queue:         LinearStrategy{},
		lastVolume:    DefaultVolume,
		metadataChan:  make(chan TrackMetadataEvent, 32),
		metadataQueue: make(chan metadataRequest, 2048),
		lyricsChan:    make(chan LyricsEvent, 32),
		lyricsQueue:   make(chan lyricsRequest, 2048),
		metadataCache: newMetadataCache(metadataCacheMaxEntries, metadataCacheMaxBytes),
		lyricsCache:   newLyricsCache(lyricsCacheMaxEntries, lyricsCacheMaxBytes),
		cmdChan:       make(chan Command, 256),
		ctx:           ctx,
		cancel:        cancel,
		playerSubs:    make(map[chan player.Event]struct{}),
		metadataSubs:  make(map[chan TrackMetadataEvent]struct{}),
		lyricsSubs:    make(map[chan LyricsEvent]struct{}),
		stateSubs:     make(map[chan StateEvent]struct{}),
	}
	app.lyricsResolver = lyrics.NewResolver(
		lyrics.NewSidecarProvider(),
		lyrics.NewEmbeddedProvider(app.readLyricsFromTagsCached),
		lyrics.NewLrcLibProvider(cfg.Lyrics.LrcLib, cfg.Cache.Dir),
	)
	app.commandWG.Go(app.commandLoop)
	go app.forwardPlayerEvents()
	go app.forwardMetadataEvents()
	go app.forwardLyricsEvents()
	for range metadataWorkerCount {
		app.metadataWG.Go(app.metadataWorker)
	}
	for range lyricsWorkerCount {
		app.lyricsWG.Go(app.lyricsWorker)
	}
	app.engine.SetVolume(DefaultVolume)

	app.shutdownWG.Go(func() {
		<-app.ctx.Done()
		app.commandWG.Wait()
		app.metadataWG.Wait()
		app.lyricsWG.Wait()
		close(app.metadataChan)
		close(app.lyricsChan)
	})

	return app
}

// Restore loads playlist and queue state without starting playback.
func (a *App) Restore(tracks []Track, cursor int, mode QueueMode) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	a.nextTrackID = 0
	a.state.Playlist = make([]Track, 0, len(tracks))
	for _, track := range tracks {
		if track.ID == 0 {
			track.ID = a.nextID()
		} else if track.ID > a.nextTrackID {
			a.nextTrackID = track.ID
		}
		a.state.Playlist = append(a.state.Playlist, track)
	}
	a.state.PlaylistErr = nil
	a.state.Playing = -1
	if cursor >= 0 && cursor < len(tracks) {
		a.state.Cursor = cursor
	} else if len(tracks) > 0 {
		a.state.Cursor = 0
	} else {
		a.state.Cursor = -1
	}
	a.resetPlaybackState()
	a.setQueueMode(mode)
}

// State returns a snapshot of the current state.
func (a *App) State() State {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return copyState(a.state)
}

// Dispatch queues a command for processing.
func (a *App) Dispatch(cmd Command) error {
	select {
	case <-a.ctx.Done():
		return ErrAppClosed
	case a.cmdChan <- cmd:
		return nil
	default:
		return ErrCommandQueueFull
	}
}

// SubscribePlayerEvents registers a new player event subscriber.
// Use the returned unsubscribe func to stop receiving events.
func (a *App) SubscribePlayerEvents() (<-chan player.Event, func()) {
	ch := make(chan player.Event, 8)
	a.subsMu.Lock()
	a.playerSubs[ch] = struct{}{}
	a.subsMu.Unlock()
	return ch, func() { a.unsubscribePlayerEvents(ch) }
}

// SubscribeMetadataEvents registers a new metadata event subscriber.
// Use the returned unsubscribe func to stop receiving events.
func (a *App) SubscribeMetadataEvents() (<-chan TrackMetadataEvent, func()) {
	ch := make(chan TrackMetadataEvent, 8)
	a.subsMu.Lock()
	a.metadataSubs[ch] = struct{}{}
	a.subsMu.Unlock()
	return ch, func() { a.unsubscribeMetadataEvents(ch) }
}

// SubscribeLyricsEvents registers a new lyrics event subscriber.
// Use the returned unsubscribe func to stop receiving events.
func (a *App) SubscribeLyricsEvents() (<-chan LyricsEvent, func()) {
	ch := make(chan LyricsEvent, 8)
	a.subsMu.Lock()
	a.lyricsSubs[ch] = struct{}{}
	a.subsMu.Unlock()
	return ch, func() { a.unsubscribeLyricsEvents(ch) }
}

// SubscribeStateEvents registers a new state event subscriber.
// Use the returned unsubscribe func to stop receiving events.
func (a *App) SubscribeStateEvents() (<-chan StateEvent, func()) {
	ch := make(chan StateEvent, 8)
	a.subsMu.Lock()
	a.stateSubs[ch] = struct{}{}
	a.subsMu.Unlock()
	return ch, func() { a.unsubscribeStateEvents(ch) }
}

// apply executes a command against the current state.
func (a *App) apply(cmd Command) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	switch cmd.Type {
	case CmdAdd:
		a.addTrack(cmd.Track)
	case CmdAddAll:
		for _, track := range cmd.Tracks {
			a.addTrack(track)
		}
	case CmdClear:
		a.clearPlaylist()
		a.engine.Stop()
	case CmdSelectUp:
		a.selectUp()
	case CmdSelectDown:
		a.selectDown()
	case CmdSelectTop:
		a.selectTop()
	case CmdSelectBottom:
		a.selectBottom()
	case CmdSelectPageUp:
		a.selectPageUp(cmd.Count)
	case CmdSelectPageDown:
		a.selectPageDown(cmd.Count)
	case CmdSelectIndex:
		a.selectIndex(cmd.Index)
	case CmdPlayFromCursor:
		index := a.playFromCursor()
		if index >= 0 {
			a.playIndex(index)
		}
	case CmdNext:
		index := a.next()
		if index >= 0 {
			a.playIndex(index)
		} else {
			a.stop()
			a.engine.Stop()
		}
	case CmdPrev:
		index := a.prev()
		if index >= 0 {
			a.playIndex(index)
		}
	case CmdStop:
		a.stop()
		a.engine.Stop()
	case CmdTogglePause:
		a.engine.TogglePause()
	case CmdRemoveAt:
		removedPlaying, removed := a.removeAt(cmd.Index)
		if removed && removedPlaying {
			a.engine.Stop()
		}
	case CmdMoveUp:
		a.moveUp()
	case CmdMoveDown:
		a.moveDown()
	case CmdSetQueueMode:
		a.setQueueMode(cmd.Mode)
	case CmdVolumeUp:
		a.adjustVolume(VolumeStep)
	case CmdVolumeDown:
		a.adjustVolume(-VolumeStep)
	case CmdToggleMute:
		a.toggleMute()
	case CmdSetVolume:
		a.setVolume(cmd.Volume)
	case CmdSeekBy:
		a.seekBy(cmd.Offset)
	case CmdRequestMetadata:
		a.requestMetadata(cmd.TrackID, cmd.Path, cmd.Scope)
	case CmdRequestLyrics:
		a.requestLyrics(cmd.TrackID, cmd.Path)
	}
}

func (a *App) commandLoop() {
	for {
		select {
		case <-a.ctx.Done():
			a.closeStateSubscribers()
			return
		case cmd := <-a.cmdChan:
			a.stateMu.RLock()
			before := copyState(a.state)
			a.stateMu.RUnlock()
			a.apply(cmd)
			a.stateMu.RLock()
			after := copyState(a.state)
			a.stateMu.RUnlock()
			changes := DiffState(before, after)
			if changes != StateChangeNone {
				a.broadcastStateEvent(StateEvent{
					Source:  StateEventCommand,
					Command: cmd,
					Changes: changes,
				})
			}
		}
	}
}

// HandlePlayerEvent updates state in response to a player event.
func (a *App) HandlePlayerEvent(event player.Event) {
	a.stateMu.RLock()
	before := copyState(a.state)
	a.stateMu.RUnlock()

	a.stateMu.Lock()
	switch event.Type {
	case player.EventTrackEnded:
		index := a.next()
		if index >= 0 {
			a.playIndex(index)
		} else {
			a.resetPlaybackState()
		}
	case player.EventTrackError:
		a.state.PlaylistErr = event.Err
		index := a.next()
		if index >= 0 {
			a.playIndex(index)
		}
	case player.EventTrackStarted:
		a.state.PlaylistErr = nil
		a.state.PlayState = PlaybackPlaying
		a.state.PlayTrack = event.Path
		a.state.PlayStart = time.Now()
		a.state.PlayDuration = event.Dur
		a.state.PausedAt = time.Time{}
		a.state.PausedFor = 0
	case player.EventTrackPaused:
		if a.state.PlayState == PlaybackPlaying {
			a.state.PlayState = PlaybackPaused
			a.state.PausedAt = time.Now()
		}
	case player.EventTrackResumed:
		if a.state.PlayState == PlaybackPaused {
			a.state.PlayState = PlaybackPlaying
			if !a.state.PausedAt.IsZero() {
				a.state.PausedFor += time.Since(a.state.PausedAt)
				a.state.PausedAt = time.Time{}
			}
		}
	case player.EventTrackStopped:
		a.resetPlaybackState()
	}
	a.stateMu.Unlock()

	a.stateMu.RLock()
	after := copyState(a.state)
	a.stateMu.RUnlock()
	changes := DiffState(before, after)
	if changes != StateChangeNone {
		a.broadcastStateEvent(StateEvent{
			Source:  StateEventPlayer,
			Changes: changes,
		})
	}
}

// Shutdown shuts down the player engine and background workers.
func (a *App) Shutdown() {
	if a.closed.Swap(true) {
		return
	}
	a.cancel()
	a.engine.Close()
}

// ShutdownAndWait shuts down the player engine and waits for background workers to stop.
func (a *App) ShutdownAndWait() {
	a.Shutdown()
	a.shutdownWG.Wait()
}

func (a *App) forwardPlayerEvents() {
	for event := range a.engine.Events() {
		a.HandlePlayerEvent(event)
		a.broadcastPlayerEvent(event)
	}
	a.subsMu.Lock()
	for ch := range a.playerSubs {
		close(ch)
		delete(a.playerSubs, ch)
	}
	a.subsMu.Unlock()
}

func (a *App) forwardMetadataEvents() {
	for event := range a.metadataChan {
		a.HandleMetadataEvent(event)
		a.broadcastMetadataEvent(event)
	}
	a.subsMu.Lock()
	for ch := range a.metadataSubs {
		close(ch)
		delete(a.metadataSubs, ch)
	}
	a.subsMu.Unlock()
}

func (a *App) forwardLyricsEvents() {
	for event := range a.lyricsChan {
		a.broadcastLyricsEvent(event)
	}
	a.subsMu.Lock()
	for ch := range a.lyricsSubs {
		close(ch)
		delete(a.lyricsSubs, ch)
	}
	a.subsMu.Unlock()
}

func (a *App) broadcastPlayerEvent(event player.Event) {
	a.subsMu.Lock()
	for ch := range a.playerSubs {
		select {
		case ch <- event:
		default:
		}
	}
	a.subsMu.Unlock()
}

func (a *App) broadcastLyricsEvent(event LyricsEvent) {
	a.subsMu.Lock()
	for ch := range a.lyricsSubs {
		select {
		case ch <- event:
		default:
		}
	}
	a.subsMu.Unlock()
}

func (a *App) broadcastMetadataEvent(event TrackMetadataEvent) {
	a.subsMu.Lock()
	for ch := range a.metadataSubs {
		select {
		case ch <- event:
		default:
		}
	}
	a.subsMu.Unlock()
}

func (a *App) broadcastStateEvent(event StateEvent) {
	a.subsMu.Lock()
	for ch := range a.stateSubs {
		select {
		case ch <- event:
		default:
		}
	}
	a.subsMu.Unlock()
}

func (a *App) unsubscribePlayerEvents(ch chan player.Event) {
	a.subsMu.Lock()
	if _, ok := a.playerSubs[ch]; ok {
		delete(a.playerSubs, ch)
		close(ch)
	}
	a.subsMu.Unlock()
}

func (a *App) unsubscribeMetadataEvents(ch chan TrackMetadataEvent) {
	a.subsMu.Lock()
	if _, ok := a.metadataSubs[ch]; ok {
		delete(a.metadataSubs, ch)
		close(ch)
	}
	a.subsMu.Unlock()
}

func (a *App) unsubscribeLyricsEvents(ch chan LyricsEvent) {
	a.subsMu.Lock()
	if _, ok := a.lyricsSubs[ch]; ok {
		delete(a.lyricsSubs, ch)
		close(ch)
	}
	a.subsMu.Unlock()
}

func (a *App) unsubscribeStateEvents(ch chan StateEvent) {
	a.subsMu.Lock()
	if _, ok := a.stateSubs[ch]; ok {
		delete(a.stateSubs, ch)
		close(ch)
	}
	a.subsMu.Unlock()
}

func copyState(s State) State {
	if len(s.Playlist) == 0 {
		return s
	}
	playlist := make([]Track, len(s.Playlist))
	copy(playlist, s.Playlist)
	s.Playlist = playlist
	return s
}

func (a *App) resetPlaybackState() {
	a.state.PlayState = PlaybackStopped
	a.state.PlayTrack = ""
	a.state.PlayStart = time.Time{}
	a.state.PlayDuration = 0
	a.state.PausedAt = time.Time{}
	a.state.PausedFor = 0
}

func (a *App) seekBy(offset time.Duration) {
	if a.state.PlayState == PlaybackStopped {
		return
	}

	elapsed := a.state.Elapsed()
	target := max(elapsed+offset, 0)

	if a.state.PlayDuration > 0 && target >= a.state.PlayDuration {
		index := a.next()
		if index >= 0 {
			a.playIndex(index)
		} else {
			a.stop()
			a.engine.Stop()
		}
		return
	}

	result := a.engine.SeekTo(target)
	if !result.Ok {
		return
	}
	pos := target
	if result.Pos > 0 {
		pos = result.Pos
	}
	if result.Dur > 0 {
		a.state.PlayDuration = result.Dur
	}

	if a.state.PlayState == PlaybackPaused && !a.state.PausedAt.IsZero() {
		a.state.PlayStart = a.state.PausedAt.Add(-a.state.PausedFor - pos)
		return
	}
	a.state.PlayStart = time.Now().Add(-a.state.PausedFor - pos)
}

func (a *App) playIndex(index int) {
	if index < 0 || index >= len(a.state.Playlist) {
		return
	}
	a.engine.Play(a.state.Playlist[index].Path)
}

func (a *App) setQueueMode(mode QueueMode) {
	if mode == a.state.QueueMode {
		return
	}
	switch mode {
	case QueueModeShuffle:
		a.queue = NewShuffleStrategy()
		a.state.QueueMode = mode
	case QueueModeRepeatOne:
		a.queue = RepeatOneStrategy{}
		a.state.QueueMode = mode
	case QueueModeRepeatAll:
		a.queue = RepeatAllStrategy{}
		a.state.QueueMode = mode
	case QueueModeStopAfterCurrent:
		a.queue = StopAfterCurrentStrategy{}
		a.state.QueueMode = mode
	default:
		a.queue = LinearStrategy{}
		a.state.QueueMode = QueueModeLinear
	}
}

func (a *App) closeStateSubscribers() {
	a.subsMu.Lock()
	for ch := range a.stateSubs {
		close(ch)
		delete(a.stateSubs, ch)
	}
	a.subsMu.Unlock()
}
