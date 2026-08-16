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

// Validate ensures audio settings are sane.
func (c AudioConfig) Validate() error {
	if c.SampleRate <= 0 {
		return fmt.Errorf("audio.sample_rate must be > 0")
	}
	if c.ResampleQuality < 1 || c.ResampleQuality > 64 {
		return fmt.Errorf("audio.resample_quality must be between 1 and 64")
	}
	if c.BufferMs <= 0 {
		return fmt.Errorf("audio.buffer_ms must be > 0")
	}
	return nil
}

// MPRISConfig holds DBus/MPRIS integration settings.
type MPRISConfig struct {
	Enabled bool `toml:"enabled" comment:"Enable DBus/MPRIS integration for media controls."`
}

// IPCConfig holds single-instance IPC settings.
type IPCConfig struct {
	SingleInstance SingleInstanceMode `toml:"single_instance" comment:"Single-instance handoff method: auto, unix-socket, or off."`
}

// Validate ensures the configured IPC mode is recognized.
func (c IPCConfig) Validate() error {
	switch c.SingleInstance {
	case SingleInstanceAuto, SingleInstanceUnixSocket, SingleInstanceOff:
		return nil
	default:
		return fmt.Errorf("ipc.single_instance must be one of auto, unix-socket, off")
	}
}

// SingleInstanceMode selects the IPC method used for single-instance handoff.
type SingleInstanceMode string

const (
	// SingleInstanceAuto uses the platform's preferred supported IPC method.
	SingleInstanceAuto SingleInstanceMode = "auto"
	// SingleInstanceUnixSocket explicitly requires Unix-socket IPC.
	SingleInstanceUnixSocket SingleInstanceMode = "unix-socket"
	// SingleInstanceOff disables single-instance IPC handoff.
	SingleInstanceOff SingleInstanceMode = "off"
)

type ThemeConfig struct {
	Preset     string `toml:"preset" comment:"Built-in theme preset."`
	Primary    string `toml:"primary,omitempty" comment:"Override the main accent color (e.g., highlighting selected items)."`
	Secondary  string `toml:"secondary,omitempty" comment:"Override the secondary accent color (e.g., search highlights, minor elements)."`
	Muted      string `toml:"muted,omitempty" comment:"Override faded or dimmed text."`
	Highlight  string `toml:"highlight,omitempty" comment:"Override the highlight color."`
	Info       string `toml:"info,omitempty" comment:"Override info indications."`
	Danger     string `toml:"danger,omitempty" comment:"Override danger indications."`
	Warning    string `toml:"warning,omitempty" comment:"Override warning indications."`
	Working    string `toml:"working,omitempty" comment:"Override working indications."`
	Background string `toml:"background,omitempty" comment:"Override the background color; use 'terminal' for the terminal default."`
	Foreground string `toml:"foreground,omitempty" comment:"Override the foreground color; use 'terminal' for the terminal default."`
}

type TUIConfig struct {
	FPS             int         `toml:"FPS" comment:"Frames per second for the terminal UI (1-120)."`
	ArtworkRenderer string      `toml:"artwork_renderer" comment:"Album artwork renderer: auto, kitty, blocks, or none."`
	ArtworkAspect   float64     `toml:"artwork_aspect" comment:"Artwork box width/height ratio for terminal cells (e.g., 2.0 looks square on most fonts)."`
	BrowserHome     string      `toml:"browser_home" comment:"Default directory of music browser"`
	Theme           ThemeConfig `toml:"theme"`
}

// Validate ensures TUI settings are sane.
func (c TUIConfig) Validate() error {
	if c.ArtworkAspect <= 0 {
		return fmt.Errorf("tui.artwork_aspect must be > 0")
	}
	switch c.ArtworkRenderer {
	case "auto", "kitty", "blocks", "none":
	default:
		return fmt.Errorf("tui.artwork_renderer must be one of auto, kitty, blocks, none")
	}
	return nil
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

// LibraryConfig holds media library settings.
type LibraryConfig struct {
	MaxArchiveMemberSize ByteSize `toml:"max_archive_member_size" comment:"Maximum decoded size of one file inside an archive."`
}

// Validate ensures library settings are sane.
func (c LibraryConfig) Validate() error {
	if c.MaxArchiveMemberSize <= 0 {
		return fmt.Errorf("library.max_archive_member_size must be > 0")
	}
	return nil
}

// Config is the root configuration object.
type Config struct {
	Audio   AudioConfig   `toml:"audio"`
	MPRIS   MPRISConfig   `toml:"mpris"`
	IPC     IPCConfig     `toml:"ipc"`
	TUI     TUIConfig     `toml:"tui"`
	Lyrics  LyricsConfig  `toml:"lyrics"`
	Cache   CacheConfig   `toml:"cache"`
	Library LibraryConfig `toml:"library"`
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
		IPC: IPCConfig{
			SingleInstance: SingleInstanceAuto,
		},
		TUI: TUIConfig{
			FPS:             60,
			ArtworkRenderer: "auto",
			ArtworkAspect:   2.0,
			BrowserHome: func() string {
				d, _ := os.UserHomeDir()
				return d
			}(),
			Theme: ThemeConfig{Preset: "default"},
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
		Library: LibraryConfig{
			MaxArchiveMemberSize: 512 * 1024 * 1024,
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := toml.Marshal(Default())
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Validate ensures configuration values are sane.
func (c Config) Validate() error {
	if err := c.Audio.Validate(); err != nil {
		return err
	}
	if err := c.IPC.Validate(); err != nil {
		return err
	}
	if err := c.TUI.Validate(); err != nil {
		return err
	}
	if err := c.Library.Validate(); err != nil {
		return err
	}
	return nil
}
