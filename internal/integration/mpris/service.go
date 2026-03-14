package mpris

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/godbus/dbus/v5"
)

const (
	serviceName = "org.mpris.MediaPlayer2.tmus"
	objectPath  = dbus.ObjectPath("/org/mpris/MediaPlayer2")

	ifaceRoot   = "org.mpris.MediaPlayer2"
	ifacePlayer = "org.mpris.MediaPlayer2.Player"
	ifaceProps  = "org.freedesktop.DBus.Properties"
)

var ErrNameOwned = errors.New("mpris name already owned")

// Service exposes the MPRIS interfaces over DBus.
type Service struct {
	conn          *dbus.Conn
	app           *core.App
	stateCh       <-chan core.StateEvent
	unsub         func()
	closed        bool
	mu            sync.Mutex
	lastPlayerMap map[string]dbus.Variant
}

// Start registers the MPRIS service if possible.
// Returns (nil, nil) if the name is already owned.
func Start(appRef *core.App) (*Service, error) {
	if appRef == nil {
		return nil, errors.New("app is nil")
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return nil, ErrNameOwned
	}

	stateCh, unsub := appRef.SubscribeStateEvents()

	svc := &Service{
		conn:          conn,
		app:           appRef,
		stateCh:       stateCh,
		unsub:         unsub,
		lastPlayerMap: make(map[string]dbus.Variant),
	}

	if err := conn.Export(svc, objectPath, ifaceRoot); err != nil {
		svc.Close()
		return nil, err
	}
	if err := conn.ExportMethodTable(playerMethodTable(svc), objectPath, ifacePlayer); err != nil {
		svc.Close()
		return nil, err
	}
	if err := conn.Export(svc, objectPath, ifaceProps); err != nil {
		svc.Close()
		return nil, err
	}

	svc.lastPlayerMap = svc.playerProperties(appRef.State())
	go svc.loop()
	return svc, nil
}

// Close releases the DBus name and stops event handling.
func (s *Service) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	if s.unsub != nil {
		s.unsub()
	}
	if s.conn != nil {
		_, _ = s.conn.ReleaseName(serviceName)
		_ = s.conn.Close()
	}
}

func (s *Service) loop() {
	for event := range s.stateCh {
		_ = event
		s.emitPlayerChanges()
		if event.Source == core.StateEventCommand && event.Command.Type == core.CmdSeekBy {
			s.emitSeeked()
		}
	}
}

func (s *Service) emitPlayerChanges() {
	state := s.app.State()
	props := s.playerProperties(state)
	s.mu.Lock()
	changed := diffProperties(s.lastPlayerMap, props)
	if len(changed) == 0 {
		s.mu.Unlock()
		return
	}
	s.lastPlayerMap = props
	s.mu.Unlock()

	_ = s.conn.Emit(objectPath, ifaceProps+".PropertiesChanged", ifacePlayer, changed, []string{})
}

func (s *Service) emitSeeked() {
	state := s.app.State()
	pos := toMicroseconds(state.Elapsed())
	_ = s.conn.Emit(objectPath, ifacePlayer+".Seeked", pos)
}

// Raise is a no-op (tmus is a terminal app).
func (s *Service) Raise() *dbus.Error {
	return nil
}

// Quit is a no-op (tmus does not quit via MPRIS).
func (s *Service) Quit() *dbus.Error {
	return nil
}

// Play starts or resumes playback.
func (s *Service) Play() *dbus.Error {
	state := s.app.State()
	switch state.PlayState {
	case core.PlaybackPaused:
		return s.dispatch(core.Command{Type: core.CmdTogglePause})
	case core.PlaybackStopped:
		return s.dispatch(core.Command{Type: core.CmdPlayFromCursor})
	default:
		return nil
	}
}

// Pause pauses playback.
func (s *Service) Pause() *dbus.Error {
	state := s.app.State()
	if state.PlayState == core.PlaybackPlaying {
		return s.dispatch(core.Command{Type: core.CmdTogglePause})
	}
	return nil
}

// PlayPause toggles playback state.
func (s *Service) PlayPause() *dbus.Error {
	state := s.app.State()
	if state.PlayState == core.PlaybackStopped {
		return s.dispatch(core.Command{Type: core.CmdPlayFromCursor})
	}
	return s.dispatch(core.Command{Type: core.CmdTogglePause})
}

// Stop stops playback.
func (s *Service) Stop() *dbus.Error {
	return s.dispatch(core.Command{Type: core.CmdStop})
}

// Next advances to the next track.
func (s *Service) Next() *dbus.Error {
	return s.dispatch(core.Command{Type: core.CmdNext})
}

// Previous goes to the previous track.
func (s *Service) Previous() *dbus.Error {
	return s.dispatch(core.Command{Type: core.CmdPrev})
}

// seek performs a relative seek in microseconds.
func (s *Service) seek(offset int64) *dbus.Error {
	if offset == 0 {
		return nil
	}
	return s.dispatch(core.Command{Type: core.CmdSeekBy, Offset: microsecondsToDuration(offset)})
}

// SetPosition seeks to an absolute position in microseconds.
func (s *Service) SetPosition(trackID dbus.ObjectPath, pos int64) *dbus.Error {
	state := s.app.State()
	currentID := currentTrackID(state)
	if currentID == 0 || trackID != trackObjectPath(currentID) {
		return nil
	}
	target := microsecondsToDuration(pos)
	offset := target - state.Elapsed()
	return s.dispatch(core.Command{Type: core.CmdSeekBy, Offset: offset})
}

// OpenUri opens a file URI or absolute path.
func (s *Service) OpenUri(uri string) *dbus.Error {
	path, err := resolveURI(uri)
	if err != nil {
		return dbus.MakeFailedError(err)
	}
	if !library.IsAudio(path) {
		return dbus.MakeFailedError(fmt.Errorf("unsupported media: %s", path))
	}
	state := s.app.State()
	index := len(state.Playlist)
	track := core.Track{Name: filepath.Base(path), Path: path}
	if err := s.dispatch(core.Command{Type: core.CmdAddAll, Tracks: []core.Track{track}}); err != nil {
		return err
	}
	if err := s.dispatch(core.Command{Type: core.CmdSelectIndex, Index: index}); err != nil {
		return err
	}
	return s.dispatch(core.Command{Type: core.CmdPlayFromCursor})
}

func playerMethodTable(svc *Service) map[string]any {
	return map[string]any{
		"Play":        svc.Play,
		"Pause":       svc.Pause,
		"PlayPause":   svc.PlayPause,
		"Stop":        svc.Stop,
		"Next":        svc.Next,
		"Previous":    svc.Previous,
		"Seek":        svc.seek,
		"SetPosition": svc.SetPosition,
		"OpenUri":     svc.OpenUri,
	}
}

// Get implements org.freedesktop.DBus.Properties.Get.
func (s *Service) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	switch iface {
	case ifaceRoot:
		value, ok := s.rootProperty(prop)
		if !ok {
			return dbus.Variant{}, dbus.MakeFailedError(errors.New("unknown property"))
		}
		return value, nil
	case ifacePlayer:
		value, ok := s.playerProperty(prop, s.app.State())
		if !ok {
			return dbus.Variant{}, dbus.MakeFailedError(errors.New("unknown property"))
		}
		return value, nil
	default:
		return dbus.Variant{}, dbus.MakeFailedError(errors.New("unknown interface"))
	}
}

// Set implements org.freedesktop.DBus.Properties.Set.
func (s *Service) Set(iface, prop string, value dbus.Variant) *dbus.Error {
	if iface != ifacePlayer {
		return dbus.MakeFailedError(errors.New("read-only interface"))
	}
	switch prop {
	case "Volume":
		vol, ok := value.Value().(float64)
		if !ok {
			return dbus.MakeFailedError(errors.New("invalid volume"))
		}
		level := min(max(int(vol*100), 0), 100)
		return s.dispatch(core.Command{Type: core.CmdSetVolume, Volume: level})
	case "Shuffle":
		shuffle, ok := value.Value().(bool)
		if !ok {
			return dbus.MakeFailedError(errors.New("invalid shuffle"))
		}
		return s.dispatch(core.Command{Type: core.CmdSetQueueMode, Mode: shuffleToQueueMode(shuffle)})
	case "LoopStatus":
		loop, ok := value.Value().(string)
		if !ok {
			return dbus.MakeFailedError(errors.New("invalid loop status"))
		}
		mode, ok := loopStatusToQueueMode(loop)
		if !ok {
			return dbus.MakeFailedError(errors.New("unsupported loop status"))
		}
		return s.dispatch(core.Command{Type: core.CmdSetQueueMode, Mode: mode})
	default:
		return dbus.MakeFailedError(errors.New("unsupported property"))
	}
}

// GetAll implements org.freedesktop.DBus.Properties.GetAll.
func (s *Service) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	switch iface {
	case ifaceRoot:
		return s.rootProperties(), nil
	case ifacePlayer:
		return s.playerProperties(s.app.State()), nil
	default:
		return nil, dbus.MakeFailedError(errors.New("unknown interface"))
	}
}

func (s *Service) dispatch(cmd core.Command) *dbus.Error {
	if err := s.app.Dispatch(cmd); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func (s *Service) rootProperties() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"CanQuit":             dbus.MakeVariant(false),
		"CanRaise":            dbus.MakeVariant(false),
		"HasTrackList":        dbus.MakeVariant(false),
		"Identity":            dbus.MakeVariant("tmus"),
		"SupportedUriSchemes": dbus.MakeVariant([]string{"file"}),
		"SupportedMimeTypes": dbus.MakeVariant([]string{
			"audio/mpeg",
			"audio/flac",
			"audio/ogg",
			"audio/wav",
		}),
	}
}

func (s *Service) rootProperty(name string) (dbus.Variant, bool) {
	props := s.rootProperties()
	value, ok := props[name]
	return value, ok
}

func (s *Service) playerProperties(state core.State) map[string]dbus.Variant {
	props := map[string]dbus.Variant{}
	maps.Copy(props, s.playerPropertyMap(state))
	return props
}

func (s *Service) playerProperty(name string, state core.State) (dbus.Variant, bool) {
	props := s.playerPropertyMap(state)
	value, ok := props[name]
	return value, ok
}

func (s *Service) playerPropertyMap(state core.State) map[string]dbus.Variant {
	metadata := s.metadataMap(state)
	status := playbackStatus(state.PlayState)
	loop := queueModeToLoopStatus(state.QueueMode)
	shuffle := queueModeToShuffle(state.QueueMode)
	canPlay := len(state.Playlist) > 0
	canPause := len(state.Playlist) > 0
	canSeek := state.PlayDuration > 0
	canGo := len(state.Playlist) > 0
	return map[string]dbus.Variant{
		"PlaybackStatus": dbus.MakeVariant(status),
		"LoopStatus":     dbus.MakeVariant(loop),
		"Shuffle":        dbus.MakeVariant(shuffle),
		"Volume":         dbus.MakeVariant(float64(state.Volume) / 100.0),
		"Position":       dbus.MakeVariant(toMicroseconds(state.Elapsed())),
		"MinimumRate":    dbus.MakeVariant(1.0),
		"MaximumRate":    dbus.MakeVariant(1.0),
		"Rate":           dbus.MakeVariant(1.0),
		"CanGoNext":      dbus.MakeVariant(canGo),
		"CanGoPrevious":  dbus.MakeVariant(canGo),
		"CanPlay":        dbus.MakeVariant(canPlay),
		"CanPause":       dbus.MakeVariant(canPause),
		"CanSeek":        dbus.MakeVariant(canSeek),
		"CanControl":     dbus.MakeVariant(true),
		"Metadata":       dbus.MakeVariant(metadata),
	}
}

func (s *Service) metadataMap(state core.State) map[string]dbus.Variant {
	metadata := map[string]dbus.Variant{}
	trackID := currentTrackID(state)
	metadata["mpris:trackid"] = dbus.MakeVariant(trackObjectPath(trackID))

	if state.PlayState == core.PlaybackStopped || trackID == 0 {
		return metadata
	}
	track := state.Playlist[state.Playing]
	title := track.Title
	if title == "" {
		title = track.Name
	}
	if title != "" {
		metadata["xesam:title"] = dbus.MakeVariant(title)
	}
	if track.Artist != "" {
		metadata["xesam:artist"] = dbus.MakeVariant([]string{track.Artist})
	}
	if track.Album != "" {
		metadata["xesam:album"] = dbus.MakeVariant(track.Album)
	}
	if state.PlayDuration > 0 {
		metadata["mpris:length"] = dbus.MakeVariant(toMicroseconds(state.PlayDuration))
	}
	if track.Path != "" {
		if uri := fileURI(track.Path); uri != "" {
			metadata["xesam:url"] = dbus.MakeVariant(uri)
		}
	}
	return metadata
}

func playbackStatus(state core.PlaybackState) string {
	switch state {
	case core.PlaybackPlaying:
		return "Playing"
	case core.PlaybackPaused:
		return "Paused"
	default:
		return "Stopped"
	}
}

func queueModeToLoopStatus(mode core.QueueMode) string {
	switch mode {
	case core.QueueModeRepeatOne:
		return "Track"
	case core.QueueModeRepeatAll:
		return "Playlist"
	default:
		return "None"
	}
}

func queueModeToShuffle(mode core.QueueMode) bool {
	return mode == core.QueueModeShuffle
}

func loopStatusToQueueMode(value string) (core.QueueMode, bool) {
	switch value {
	case "None":
		return core.QueueModeLinear, true
	case "Track":
		return core.QueueModeRepeatOne, true
	case "Playlist":
		return core.QueueModeRepeatAll, true
	default:
		return core.QueueModeLinear, false
	}
}

func shuffleToQueueMode(shuffle bool) core.QueueMode {
	if shuffle {
		return core.QueueModeShuffle
	}
	return core.QueueModeLinear
}

func diffProperties(prev, next map[string]dbus.Variant) map[string]dbus.Variant {
	changed := make(map[string]dbus.Variant)
	for key, nextVal := range next {
		prevVal, ok := prev[key]
		if !ok || !equalVariant(prevVal, nextVal) {
			changed[key] = nextVal
		}
	}
	return changed
}

func equalVariant(a, b dbus.Variant) bool {
	return reflect.DeepEqual(a.Value(), b.Value())
}

func toMicroseconds(dur time.Duration) int64 {
	return dur.Nanoseconds() / 1000
}

func microsecondsToDuration(value int64) time.Duration {
	return time.Duration(value) * time.Microsecond
}

func currentTrackID(state core.State) uint64 {
	if state.Playing < 0 || state.Playing >= len(state.Playlist) {
		return 0
	}
	return state.Playlist[state.Playing].ID
}

func trackObjectPath(id uint64) dbus.ObjectPath {
	return dbus.ObjectPath(fmt.Sprintf("/org/mpris/MediaPlayer2/track/%d", id))
}

func resolveURI(value string) (string, error) {
	if value == "" {
		return "", errors.New("uri is empty")
	}
	if strings.HasPrefix(value, "file://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		path, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", err
		}
		if path == "" {
			return "", errors.New("uri path is empty")
		}
		return path, nil
	}
	if filepath.IsAbs(value) {
		return value, nil
	}
	return "", errors.New("unsupported uri")
}

func fileURI(path string) string {
	if path == "" {
		return ""
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
