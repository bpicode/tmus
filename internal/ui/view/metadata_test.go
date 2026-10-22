package view_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bpicode/tmus/internal/app/core"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

func TestModelDispatchesTrackMetadataRequest(t *testing.T) {
	m, app := newModelTest(t)
	app.Restore([]core.Track{{
		Path: filepath.Join("testdata", "Metalguy-ca - Master of Carpets.mp3"),
		Name: "Queued track",
	}}, 0, core.QueueModeLinear)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
	tui := &tuiTest{t: t, appRef: app, tm: tm}
	tui.waitForOutput("Queued track")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	tui.waitForOutput("MoC Deluxe Edition")
	tm.Type("q")
	tui.waitFinished()
}

func TestModelReportsTrackMetadataDispatchFailure(t *testing.T) {
	m, app := newModelTest(t)
	app.Restore([]core.Track{{Path: "/music/one.flac", Name: "Queued track"}}, 0, core.QueueModeLinear)
	observed := &observedModel{Model: m}
	tm := teatest.NewTestModel(t, observed, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
	tui := &tuiTest{t: t, appRef: app, tm: tm}
	tui.waitForOutput("Queued track")
	app.ShutdownAndWait()
	tm.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	observed.waitForView(t, func(content string) bool {
		return strings.Contains(content, "app is closed") && !strings.Contains(content, "Loading...")
	})
	tui.waitForOutput("❌")
	tm.Type("q")
	tui.waitFinished()
}
