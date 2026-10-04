package view_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/bpicode/tmus/internal/app/core"
	_ "github.com/bpicode/tmus/testing"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

type tuiTest struct {
	t      *testing.T
	appRef *core.App
	tm     *teatest.TestModel
}

func (tui *tuiTest) waitForOutput(values ...string) {
	tui.t.Helper()
	teatest.WaitFor(
		tui.t,
		tui.tm.Output(),
		func(output []byte) bool {
			for _, value := range values {
				if !bytes.Contains(output, []byte(value)) {
					return false
				}
			}
			return true
		},
		teatest.WithDuration(2*time.Second),
		teatest.WithCheckInterval(10*time.Millisecond),
	)
}

func (tui *tuiTest) waitForState(condition func(core.State) bool) core.State {
	tui.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := tui.appRef.State()
		if condition(state) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	state := tui.appRef.State()
	tui.t.Fatalf("condition not met after 2s; last application state: %+v", state)
	return state
}

func (tui *tuiTest) waitFinished() {
	tui.t.Helper()
	tui.tm.WaitFinished(tui.t, teatest.WithFinalTimeout(time.Second))
}
