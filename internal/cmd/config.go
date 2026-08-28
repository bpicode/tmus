package cmd

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/bpicode/tmus/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration files",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a default config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("config path: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")
		if err := config.WriteDefault(configPath, force); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote config to %s\n", configPath)
		return err
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the effective configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfigFromFlags(cmd)
		if err != nil {
			return err
		}

		data, err := toml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return err
	},
}

func init() {
	configInitCmd.Flags().Bool("force", false, "overwrite existing config file")
	configCmd.AddCommand(configInitCmd)
	addConfigFlags(configShowCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
