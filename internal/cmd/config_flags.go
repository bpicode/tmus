package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bpicode/tmus/internal/config"
)

func addConfigFlags(cmd *cobra.Command) {
	cmd.Flags().String("config", "", "path to config file")
	cmd.Flags().Int("sample-rate", 0, "override audio sample rate (Hz)")
	cmd.Flags().Int("resample-quality", 0, "override resample quality (1-64; higher uses more CPU)")
	cmd.Flags().Int("buffer-ms", 0, "override speaker buffer size in milliseconds")
	cmd.Flags().Bool("mpris", true, "enable DBus/MPRIS media controls")
	cmd.Flags().Int("fps", 0, "frames per second for the terminal UI (1-120)")
}

func loadConfigFromFlags(cmd *cobra.Command) (config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	configSet := cmd.Flags().Changed("config")
	if configSet && configPath == "" {
		return config.Config{}, fmt.Errorf("config path: empty")
	}
	if !configSet && configPath == "" {
		path, err := config.DefaultPath()
		if err != nil {
			return config.Config{}, fmt.Errorf("config path: %w", err)
		}
		configPath = path
	}
	if configSet {
		if _, err := os.Stat(configPath); err != nil {
			return config.Config{}, fmt.Errorf("config path: %w", err)
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return config.Config{}, fmt.Errorf("load config: %w", err)
	}

	if sampleRate, _ := cmd.Flags().GetInt("sample-rate"); cmd.Flags().Changed("sample-rate") {
		if sampleRate <= 0 {
			return config.Config{}, fmt.Errorf("sample-rate must be > 0")
		}
		cfg.Audio.SampleRate = sampleRate
	}
	if resampleQuality, _ := cmd.Flags().GetInt("resample-quality"); cmd.Flags().Changed("resample-quality") {
		if resampleQuality < 1 || resampleQuality > 64 {
			return config.Config{}, fmt.Errorf("resample-quality must be between 1 and 64")
		}
		cfg.Audio.ResampleQuality = resampleQuality
	}
	if bufferMs, _ := cmd.Flags().GetInt("buffer-ms"); cmd.Flags().Changed("buffer-ms") {
		if bufferMs <= 0 {
			return config.Config{}, fmt.Errorf("buffer-ms must be > 0")
		}
		cfg.Audio.BufferMs = bufferMs
	}
	if mprisEnabled, _ := cmd.Flags().GetBool("mpris"); cmd.Flags().Changed("mpris") {
		cfg.MPRIS.Enabled = mprisEnabled
	}

	if tuiFps, _ := cmd.Flags().GetInt("fps"); cmd.Flags().Changed("fps") {
		cfg.TUI.FPS = tuiFps
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}
