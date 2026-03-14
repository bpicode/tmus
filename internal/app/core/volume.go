package core

const (
	// VolumeMin is the minimum volume level.
	VolumeMin = 0
	// VolumeMax is the maximum volume level.
	VolumeMax = 100
	// VolumeStep is the delta used when adjusting volume with +/-.
	VolumeStep = 5
)

// SetVolume sets the current volume (0-100).
func (a *App) SetVolume(volume int) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	a.setVolume(volume)
}

func (a *App) adjustVolume(delta int) {
	volume := a.state.Volume
	if volume == 0 && delta > 0 && a.lastVolume > 0 {
		volume = a.lastVolume
	}
	a.setVolume(volume + delta)
}

func (a *App) toggleMute() {
	if a.state.Volume == 0 {
		restore := a.lastVolume
		if restore <= 0 {
			restore = DefaultVolume
		}
		a.setVolume(restore)
		return
	}
	a.setVolume(0)
}

func (a *App) setVolume(volume int) {
	volume = clamp(volume, VolumeMin, VolumeMax)
	a.state.Volume = volume
	if volume > 0 {
		a.lastVolume = volume
	}
	a.engine.SetVolume(volume)
}
