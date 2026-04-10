package view

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/pelletier/go-toml/v2"
)

// State captures UI-related state persisted across runs.
type State struct {
	Focus   string  `toml:"focus"`
	Browser Browser `toml:"browser"`
	Player  Player  `toml:"player"`
	Lyrics  Lyrics  `toml:"lyrics"`
}

// Browser captures persisted state for the file browser.
type Browser struct {
	Hidden bool   `toml:"hidden"`
	Cwd    string `toml:"cwd"`
}

// Player captures persisted state for the player.
type Player struct {
	Volume    *int    `toml:"volume"`
	QueueMode string  `toml:"queue_mode"`
	Playing   int     `toml:"playing"`
	Cursor    int     `toml:"cursor"`
	Playlist  []Track `toml:"playlist"`
}

// Lyrics captures persisted state for the lyrics view.
type Lyrics struct {
	FollowLine bool `toml:"follow_line"`
}

// Track captures persisted metadata for a playlist entry.
type Track struct {
	Path     string        `toml:"path"`
	Name     string        `toml:"name"`
	Artist   string        `toml:"artist"`
	Title    string        `toml:"title"`
	Album    string        `toml:"album"`
	Duration time.Duration `toml:"duration"`
}

const (
	queueModeLinear           = "linear"
	queueModeShuffle          = "shuffle"
	queueModeRepeatOne        = "repeat-one"
	queueModeRepeatAll        = "repeat-all"
	queueModeStopAfterCurrent = "stop-after-current"
)

// DefaultPath returns the default state file path.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tmus", "state.toml"), nil
}

// Load reads state from path. Missing files return a zero-value State.
func Load(path string) (State, error) {
	var s State
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return s, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := toml.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// Save writes state to path.
func Save(path string, s State) error {
	if path == "" {
		return fmt.Errorf("state path is empty")
	}
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return fmt.Errorf("state path missing directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// QueueModeString converts a queue mode into a persisted value.
func QueueModeString(mode core.QueueMode) string {
	switch mode {
	case core.QueueModeShuffle:
		return queueModeShuffle
	case core.QueueModeRepeatOne:
		return queueModeRepeatOne
	case core.QueueModeRepeatAll:
		return queueModeRepeatAll
	case core.QueueModeStopAfterCurrent:
		return queueModeStopAfterCurrent
	default:
		return queueModeLinear
	}
}

// ParseQueueMode converts a persisted value into a queue mode.
func ParseQueueMode(value string) core.QueueMode {
	switch value {
	case queueModeShuffle:
		return core.QueueModeShuffle
	case queueModeRepeatOne:
		return core.QueueModeRepeatOne
	case queueModeRepeatAll:
		return core.QueueModeRepeatAll
	case queueModeStopAfterCurrent:
		return core.QueueModeStopAfterCurrent
	default:
		return core.QueueModeLinear
	}
}

func loadState() (State, error) {
	statePath, err := DefaultPath()
	if err != nil {
		return State{}, err
	}
	return Load(statePath)
}
