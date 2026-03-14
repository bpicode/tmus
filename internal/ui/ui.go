package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/bpicode/tmus/internal/config"
	"github.com/bpicode/tmus/internal/ui/view"
)

// Run starts the TUI program.
func Run(appRef *core.App, startDir string, cfg config.Config, openFiles []string) error {
	m := view.NewModel(appRef, startDir, openFiles, cfg.TUI)
	final, err := tea.NewProgram(
		m,
		tea.WithFPS(cfg.TUI.FPS),
	).Run()
	if err != nil {
		return err
	}
	if finalModel, ok := final.(*view.Model); ok {
		if err := finalModel.SaveState(); err != nil {
			return err
		}
	}
	return nil
}
