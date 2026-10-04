package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/theme"
	"github.com/bpicode/tmus/internal/ui/view"
)

// Run creates and starts the TUI, shuts down the player, and saves its state.
func Run(appRef *core.App, startDir string, cfg config.TUIConfig, openFiles []string) error {
	m, err := view.NewModel(appRef, startDir, openFiles, cfg, theme.Resolve(cfg.Theme))
	if err != nil {
		appRef.ShutdownAndWait()
		return fmt.Errorf("create view: %w", err)
	}
	_, err = tea.NewProgram(m, tea.WithFPS(cfg.FPS)).Run()
	m.Shutdown()
	appRef.ShutdownAndWait()
	if err != nil {
		return err
	}
	return m.SaveState()
}
