package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bpicode/tmus/internal/config"
)

func addConfigFlags(cmd *cobra.Command) {
	cmd.Flags().String("config", "", "path to config file")
}

func loadConfig(cmd *cobra.Command) (config.Config, error) {
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

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}
