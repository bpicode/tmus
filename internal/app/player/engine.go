package player

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

type EventType string

const (
	EventTrackStarted EventType = "track_started"
	EventTrackEnded   EventType = "track_ended"
	EventTrackError   EventType = "track_error"
	EventTrackPaused  EventType = "track_paused"
	EventTrackResumed EventType = "track_resumed"
	EventTrackStopped EventType = "track_stopped"
)

type Event struct {
	Type EventType
	Path string
	Err  error
	Dur  time.Duration
}

type commandType string

const (
	cmdPlay        commandType = "play"
	cmdStop        commandType = "stop"
	cmdTogglePause commandType = "toggle_pause"
	cmdSeek        commandType = "seek"
	cmdQuit        commandType = "quit"
)

type command struct {
	kind commandType
	path string
	pos  time.Duration
	resp chan SeekResult
}

// Options configures the audio engine.
type Options struct {
	SampleRate      int
	ResampleQuality int
	BufferDuration  time.Duration
}

// Engine runs audio playback in a background goroutine.
type Engine struct {
	cmdCh   chan command
	eventCh chan Event

	mu          sync.Mutex
	streamer    beep.StreamSeekCloser
	ctrl        *beep.Ctrl
	volume      *effects.Volume
	sampleRate  beep.SampleRate
	targetRate  beep.SampleRate
	resampleQ   int
	bufferDur   time.Duration
	closed      atomic.Bool
	playID      atomic.Uint64
	paused      bool
	current     string
	volumeLevel int
	sourceRate  beep.SampleRate
	trackDur    time.Duration
}

func NewEngine(opts Options) *Engine {
	e := &Engine{
		cmdCh:      make(chan command, 8),
		eventCh:    make(chan Event, 8),
		targetRate: beep.SampleRate(opts.SampleRate),
		resampleQ:  opts.ResampleQuality,
		bufferDur:  opts.BufferDuration,
	}
	go e.loop()
	return e
}

func (e *Engine) Events() <-chan Event {
	return e.eventCh
}

func (e *Engine) Play(path string) {
	if e.closed.Load() {
		return
	}
	e.cmdCh <- command{kind: cmdPlay, path: path}
}

func (e *Engine) Stop() {
	if e.closed.Load() {
		return
	}
	e.cmdCh <- command{kind: cmdStop}
}

func (e *Engine) TogglePause() {
	if e.closed.Load() {
		return
	}
	e.cmdCh <- command{kind: cmdTogglePause}
}

func (e *Engine) SeekTo(pos time.Duration) SeekResult {
	if e.closed.Load() {
		return SeekResult{}
	}
	resp := make(chan SeekResult, 1)
	e.cmdCh <- command{kind: cmdSeek, pos: pos, resp: resp}
	return <-resp
}

func (e *Engine) Close() {
	if e.closed.Swap(true) {
		return
	}
	e.cmdCh <- command{kind: cmdQuit}
}

// SetVolume sets the playback volume (0-100).
func (e *Engine) SetVolume(level int) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	e.mu.Lock()
	e.volumeLevel = level
	vol := e.volume
	e.mu.Unlock()

	if vol == nil {
		return
	}

	speaker.Lock()
	applyVolume(vol, level)
	speaker.Unlock()
}

func (e *Engine) loop() {
	for cmd := range e.cmdCh {
		switch cmd.kind {
		case cmdPlay:
			e.playPath(cmd.path)
		case cmdStop:
			e.invalidatePlayback()
			e.stopCurrent()
		case cmdTogglePause:
			e.togglePause()
		case cmdSeek:
			result := e.seekTo(cmd.pos)
			cmd.resp <- result
		case cmdQuit:
			e.invalidatePlayback()
			e.stopCurrent()
			close(e.eventCh)
			return
		}
	}
}

func (e *Engine) playPath(path string) {
	if e.closed.Load() {
		return
	}
	playID := e.nextPlaybackID()
	e.stopCurrent()

	streamer, format, err := decodeFile(path)
	if err != nil {
		e.sendEvent(Event{Type: EventTrackError, Path: path, Err: err})
		return
	}

	if e.closed.Load() {
		_ = streamer.Close()
		return
	}

	if e.sampleRate == 0 {
		targetRate := e.targetRate
		buffer := targetRate.N(e.bufferDur)
		if err := speaker.Init(targetRate, buffer); err != nil {
			_ = streamer.Close()
			e.sendEvent(Event{Type: EventTrackError, Path: path, Err: fmt.Errorf("init speaker: %w", err)})
			return
		}
		e.sampleRate = targetRate
	}

	trackDur := durationFor(streamer, format)

	e.mu.Lock()
	e.streamer = streamer
	playStreamer := beep.Streamer(streamer)
	if format.SampleRate != e.sampleRate {
		playStreamer = beep.Resample(e.resampleQ, format.SampleRate, e.sampleRate, streamer)
	}
	e.sourceRate = format.SampleRate
	e.trackDur = trackDur
	vol := &effects.Volume{Streamer: playStreamer}
	applyVolume(vol, e.volumeLevel)
	e.volume = vol
	e.ctrl = &beep.Ctrl{Streamer: vol, Paused: false}
	e.paused = false
	e.current = path
	e.mu.Unlock()

	e.sendEvent(Event{Type: EventTrackStarted, Path: path, Dur: trackDur})

	speaker.Play(beep.Seq(e.ctrl, beep.Callback(func() {
		_ = streamer.Close()
		if e.playID.Load() != playID {
			return
		}
		e.sendEvent(Event{Type: EventTrackEnded, Path: path})
	})))
}

func durationFor(streamer beep.Streamer, format beep.Format) time.Duration {
	if format.SampleRate <= 0 {
		return 0
	}
	length, ok := streamer.(interface{ Len() int })
	if !ok {
		return 0
	}
	frames := length.Len()
	if frames <= 0 {
		return 0
	}
	return format.SampleRate.D(frames)
}

func (e *Engine) stopCurrent() {
	e.mu.Lock()
	streamer := e.streamer
	current := e.current
	e.ctrl = nil
	e.streamer = nil
	e.volume = nil
	e.current = ""
	e.paused = false
	e.sourceRate = 0
	e.trackDur = 0
	e.mu.Unlock()

	speaker.Clear()
	if streamer != nil {
		_ = streamer.Close()
	}
	if current != "" {
		e.sendEvent(Event{Type: EventTrackStopped, Path: current})
	}
}

func (e *Engine) togglePause() {
	e.mu.Lock()
	ctrl := e.ctrl
	current := e.current
	e.mu.Unlock()

	if ctrl == nil {
		return
	}

	speaker.Lock()
	ctrl.Paused = !ctrl.Paused
	speaker.Unlock()

	e.mu.Lock()
	e.paused = ctrl.Paused
	e.mu.Unlock()

	if ctrl.Paused {
		e.sendEvent(Event{Type: EventTrackPaused, Path: current})
	} else {
		e.sendEvent(Event{Type: EventTrackResumed, Path: current})
	}
}

type SeekResult struct {
	Ok  bool
	Pos time.Duration
	Dur time.Duration
}

func (e *Engine) seekTo(pos time.Duration) SeekResult {
	e.mu.Lock()
	streamer := e.streamer
	rate := e.sourceRate
	dur := e.trackDur
	e.mu.Unlock()

	if streamer == nil || rate <= 0 {
		return SeekResult{Ok: false}
	}

	if pos < 0 {
		pos = 0
	}

	type seeker interface {
		Seek(int) error
		Len() int
	}
	s, ok := streamer.(seeker)
	if !ok {
		return SeekResult{Ok: false}
	}

	frames := max(rate.N(pos), 0)
	if length := s.Len(); length > 0 {
		if frames >= length {
			return SeekResult{Ok: true, Dur: dur}
		}
	}

	speaker.Lock()
	err := s.Seek(frames)
	speaker.Unlock()
	if err != nil {
		return SeekResult{Ok: false}
	}

	return SeekResult{Ok: true, Pos: rate.D(frames), Dur: dur}
}

func (e *Engine) sendEvent(event Event) {
	if e.closed.Load() {
		return
	}
	select {
	case e.eventCh <- event:
	default:
	}
}

func (e *Engine) nextPlaybackID() uint64 {
	return e.playID.Add(1)
}

func applyVolume(vol *effects.Volume, level int) {
	if level <= 0 {
		vol.Silent = true
		vol.Volume = 0
		vol.Base = 2
		return
	}
	vol.Silent = false
	vol.Base = 2
	vol.Volume = volumeToExponent(level)
}

func volumeToExponent(level int) float64 {
	if level <= 1 {
		return -3
	}
	minExp := -3.0
	return minExp + (float64(level-1) * (0 - minExp) / 99)
}

func (e *Engine) invalidatePlayback() {
	e.playID.Add(1)
}
