package core

import (
	"time"

	"github.com/bpicode/tmus/internal/app/player"
)

// SpectrumBandCount is the number of frequency bands in a Spectrum.
const SpectrumBandCount = player.SpectrumBandCount

// Spectrum is an ephemeral frequency-spectrum snapshot for active playback.
// It is intentionally separate from State because consumers poll it much more
// frequently than durable application state changes. Bands are normalized to
// 0..1 and ordered from bass to treble.
type Spectrum struct {
	// Bands contains normalized bass-to-treble energy values.
	Bands [SpectrumBandCount]float64
	// PlaybackID identifies the active track represented by Bands.
	PlaybackID uint64
	// Generation changes when analysis restarts after a discontinuity.
	Generation uint64
	// Sequence allows pollers to recognize a newly published snapshot.
	Sequence uint64
	// EndFrame identifies the end of the current generation's output-rate window.
	EndFrame uint64
	// AnalyzedAt records when spectrum processing completed.
	AnalyzedAt time.Time
}

// Spectrum returns the latest analyzer result for the active playback only.
// Results from a stopped or superseded playback are represented by the zero value.
func (a *App) Spectrum() Spectrum {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	activePlaybackID := a.activePlaybackID
	if activePlaybackID == 0 {
		return Spectrum{}
	}

	snapshot := a.engine.Spectrum()
	if snapshot.PlaybackID != activePlaybackID {
		return Spectrum{}
	}
	return Spectrum{
		Bands:      snapshot.Bands,
		PlaybackID: snapshot.PlaybackID,
		Generation: snapshot.Generation,
		Sequence:   snapshot.Sequence,
		EndFrame:   snapshot.EndFrame,
		AnalyzedAt: snapshot.AnalyzedAt,
	}
}
