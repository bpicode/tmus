package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// AudioConfig holds audio-related settings.
type AudioConfig struct {
	SampleRate      int `toml:"sample_rate" comment:"Output sample rate in Hz. 44100 is a safe default."`
	ResampleQuality int `toml:"resample_quality" comment:"Resample quality 1-64 used only when source sample rate differs from output. 1-2: very fast/low quality; 3-4: good balance; >6: offline/CPU-heavy. Sane values are usually <16."`
	BufferMs        int `toml:"buffer_ms" comment:"Speaker buffer size in milliseconds (lower = lower latency, higher = more stable)."`
}

// MPRISConfig holds DBus/MPRIS integration settings.
type MPRISConfig struct {
	Enabled bool `toml:"enabled" comment:"Enable DBus/MPRIS integration for media controls."`
}

type TUIConfig struct {
	FPS           int     `toml:"FPS" comment:"Frames per second for the terminal UI (1-120)."`
	ArtworkAspect float64 `toml:"artwork_aspect" comment:"Artwork box width/height ratio for terminal cells (e.g., 2.0 looks square on most fonts)."`
}

type LyricsConfig struct {
	LrcLib LrcLibConfig `toml:"lrclib"`
}

type LrcLibCacheConfig struct {
	Enabled bool `toml:"enabled" comment:"Enable on-disk cache for lyrics"`
}

type LrcLibConfig struct {
	Enabled bool              `toml:"enabled" comment:"Enable lrclib.net integration for obtaining lyrics."`
	Cache   LrcLibCacheConfig `toml:"cache"`
}

type CacheConfig struct {
	Dir string `toml:"dir" comment:"Base directory where cache files are stored"`
}

// Config is the root configuration object.
type Config struct {
	Audio  AudioConfig  `toml:"audio"`
	MPRIS  MPRISConfig  `toml:"mpris"`
	TUI    TUIConfig    `toml:"tui"`
	Lyrics LyricsConfig `toml:"lyrics"`
	Cache  CacheConfig  `toml:"cache"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{
		Audio: AudioConfig{
			SampleRate:      44100,
			ResampleQuality: 4,
			BufferMs:        100,
		},
		MPRIS: MPRISConfig{
			Enabled: true,
		},
		TUI: TUIConfig{
			FPS:           60,
			ArtworkAspect: 2.0,
		},
		Lyrics: LyricsConfig{
			LrcLib: LrcLibConfig{
				Enabled: true,
				Cache: LrcLibCacheConfig{
					Enabled: true,
				},
			},
		},
		Cache: CacheConfig{
			Dir: func() string {
				d, err := os.UserCacheDir()
				if err != nil {
					d = os.TempDir()
				}
				return filepath.Join(d, "tmus")
			}(),
		},
	}
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tmus", "config.toml"), nil
}

// Load reads a TOML config from path. Missing files return defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// WriteDefault writes a default config to path.
func WriteDefault(path string, force bool) error {
	if path == "" {
		return fmt.Errorf("config path is empty")
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists (use --force to overwrite)")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return fmt.Errorf("config path missing directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(Default())
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Validate ensures configuration values are sane.
func (c Config) Validate() error {
	if c.Audio.SampleRate <= 0 {
		return fmt.Errorf("audio.sample_rate must be > 0")
	}
	if c.Audio.ResampleQuality < 1 || c.Audio.ResampleQuality > 64 {
		return fmt.Errorf("audio.resample_quality must be between 1 and 64")
	}
	if c.Audio.BufferMs <= 0 {
		return fmt.Errorf("audio.buffer_ms must be > 0")
	}
	if c.TUI.ArtworkAspect <= 0 {
		return fmt.Errorf("tui.artwork_aspect must be > 0")
	}
	return nil
}
