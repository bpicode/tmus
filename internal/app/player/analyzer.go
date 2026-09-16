package player

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep/v2"
	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	// SpectrumBandCount is the number of frequency bands exposed to consumers.
	SpectrumBandCount = 16

	spectrumWindowSize   = 2048
	spectrumHopSize      = spectrumWindowSize / 2
	spectrumMinHz        = 50.0
	spectrumMaxHz        = 16000.0
	spectrumFloorDB      = -60.0
	spectrumSilencePeak  = 1e-5
	spectrumWorkBufCount = 4
)

// SpectrumSnapshot is the latest completed, unsmoothed spectrum measurement.
// A zero-value snapshot means that no analyzed playback is currently active.
type SpectrumSnapshot struct {
	// Bands contains bass-to-treble energy mapped from -60..0 dBFS to 0..1.
	Bands [SpectrumBandCount]float64
	// PlaybackID identifies the track whose samples produced Bands.
	PlaybackID uint64
	// Generation changes when samples become discontinuous, such as after a seek.
	Generation uint64
	// Sequence changes whenever the published snapshot changes.
	Sequence uint64
	// EndFrame is the output-frame count at the end of the current generation's window.
	EndFrame uint64
	// AnalyzedAt records when the worker completed the FFT.
	AnalyzedAt time.Time
}

type spectrumWorkBuffer struct {
	samples [spectrumWindowSize][2]float64
}

type spectrumWork struct {
	buffer     *spectrumWorkBuffer
	playbackID uint64
	generation uint64
	endFrame   uint64
}

// spectrumAnalyzer owns the bounded handoff, FFT worker, and latest result.
// Only the speaker goroutine mutates the rolling window. Engine lifecycle
// methods resetGeneration it while the speaker is stopped or locked.
type spectrumAnalyzer struct {
	sampleRate float64

	window         [spectrumWindowSize][2]float64
	windowLen      int
	capturedFrames uint64

	freeBuffers chan *spectrumWorkBuffer
	workQueue   chan spectrumWork

	playbackID     atomic.Uint64
	generation     atomic.Uint64
	droppedWindows atomic.Uint64
	closed         atomic.Bool

	resultMu sync.RWMutex
	latest   SpectrumSnapshot

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newSpectrumAnalyzer(sampleRate beep.SampleRate) *spectrumAnalyzer {
	ctx, cancel := context.WithCancel(context.Background())
	a := &spectrumAnalyzer{
		sampleRate:  float64(sampleRate),
		freeBuffers: make(chan *spectrumWorkBuffer, spectrumWorkBufCount),
		workQueue:   make(chan spectrumWork, spectrumWorkBufCount),
		ctx:         ctx,
		cancel:      cancel,
	}
	for range spectrumWorkBufCount {
		a.freeBuffers <- new(spectrumWorkBuffer)
	}
	a.wg.Go(a.run)
	return a
}

// wrapStreamer inserts sample captureSamples without changing the stream's output or status.
func (a *spectrumAnalyzer) wrapStreamer(streamer beep.Streamer) beep.Streamer {
	return &spectrumStreamer{Streamer: streamer, analyzer: a}
}

// startPlayback begins a new playback generation. The speaker must not be pulling from
// a previously wrapped streamer while this method resets the rolling window.
func (a *spectrumAnalyzer) startPlayback(playbackID uint64) {
	a.playbackID.Store(playbackID)
	generation := a.generation.Add(1)
	a.resetCaptureState()
	a.discardQueuedWork()
	a.publishEmptySnapshot(playbackID, generation)
}

// resetGeneration starts a new analysis generation within the current playback. Callers
// hold the speaker lock so captureSamples cannot mutate the rolling window concurrently.
func (a *spectrumAnalyzer) resetGeneration(playbackID uint64) {
	if playbackID == 0 || a.playbackID.Load() != playbackID {
		return
	}
	generation := a.generation.Add(1)
	a.resetCaptureState()
	a.discardQueuedWork()
	a.publishEmptySnapshot(playbackID, generation)
}

// invalidatePlayback rejects queued and in-flight workQueue after playback has stopped.
func (a *spectrumAnalyzer) invalidatePlayback() {
	a.playbackID.Store(0)
	generation := a.generation.Add(1)
	a.resetCaptureState()
	a.discardQueuedWork()
	a.publishEmptySnapshot(0, generation)
}

func (a *spectrumAnalyzer) resetCaptureState() {
	clear(a.window[:])
	a.windowLen = 0
	a.capturedFrames = 0
}

func (a *spectrumAnalyzer) discardQueuedWork() {
	for {
		select {
		case work := <-a.workQueue:
			a.freeBuffers <- work.buffer
		default:
			return
		}
	}
}

func (a *spectrumAnalyzer) captureSamples(samples [][2]float64) {
	if len(samples) == 0 || a.closed.Load() || a.playbackID.Load() == 0 {
		return
	}

	for len(samples) > 0 {
		remainingWindowSpace := spectrumWindowSize - a.windowLen
		numSamplesToCopy := min(len(samples), remainingWindowSpace)
		copy(a.window[a.windowLen:a.windowLen+numSamplesToCopy], samples[:numSamplesToCopy])
		a.windowLen += numSamplesToCopy
		a.capturedFrames += uint64(numSamplesToCopy)
		samples = samples[numSamplesToCopy:]

		if a.windowLen < spectrumWindowSize {
			continue
		}
		a.enqueueWindow()
		copy(a.window[:spectrumHopSize], a.window[spectrumHopSize:])
		a.windowLen = spectrumHopSize
	}
}

// enqueueWindow is called on the speaker goroutine. Both pool acquisition and
// worker handoff are deliberately non-blocking: visualization capturedFrames may be
// droppedWindows, but audio samples must never be delayed.
func (a *spectrumAnalyzer) enqueueWindow() {
	var buffer *spectrumWorkBuffer
	select {
	case buffer = <-a.freeBuffers:
	default:
		a.droppedWindows.Add(1)
		return
	}

	copy(buffer.samples[:], a.window[:])
	work := spectrumWork{
		buffer:     buffer,
		playbackID: a.playbackID.Load(),
		generation: a.generation.Load(),
		endFrame:   a.capturedFrames,
	}
	select {
	case a.workQueue <- work:
	default:
		a.droppedWindows.Add(1)
		a.freeBuffers <- buffer
	}
}

func (a *spectrumAnalyzer) latestSnapshot() SpectrumSnapshot {
	a.resultMu.RLock()
	defer a.resultMu.RUnlock()
	return a.latest
}

func (a *spectrumAnalyzer) publishEmptySnapshot(playbackID, generation uint64) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	a.latest = SpectrumSnapshot{
		PlaybackID: playbackID,
		Generation: generation,
		Sequence:   a.latest.Sequence + 1,
	}
}

func (a *spectrumAnalyzer) publishSpectrum(work spectrumWork, bands [SpectrumBandCount]float64) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.playbackID.Load() != work.playbackID || a.generation.Load() != work.generation {
		return
	}
	a.latest = SpectrumSnapshot{
		Bands:      bands,
		PlaybackID: work.playbackID,
		Generation: work.generation,
		Sequence:   a.latest.Sequence + 1,
		EndFrame:   work.endFrame,
		AnalyzedAt: time.Now(),
	}
}

func (a *spectrumAnalyzer) close() {
	if a.closed.Swap(true) {
		return
	}
	a.cancel()
	a.wg.Wait()
}

func (a *spectrumAnalyzer) run() {
	worker := newSpectrumWorker(a.sampleRate)
	for {
		select {
		case <-a.ctx.Done():
			return
		case work := <-a.workQueue:
			bands := worker.analyze(work.buffer.samples[:])
			a.publishSpectrum(work, bands)
			a.freeBuffers <- work.buffer
		}
	}
}

type spectrumStreamer struct {
	beep.Streamer
	analyzer *spectrumAnalyzer
}

func (s *spectrumStreamer) Stream(samples [][2]float64) (int, bool) {
	n, ok := s.Streamer.Stream(samples)
	s.analyzer.captureSamples(samples[:n])
	return n, ok
}

type spectrumWorker struct {
	sampleRate  float64
	fft         *fourier.FFT
	window      [spectrumWindowSize]float64
	windowPower float64
	left        [spectrumWindowSize]float64
	right       [spectrumWindowSize]float64
	leftCoeff   [spectrumWindowSize/2 + 1]complex128
	rightCoeff  [spectrumWindowSize/2 + 1]complex128
	powers      [spectrumWindowSize/2 + 1]float64
	edges       [SpectrumBandCount + 1]float64
}

func newSpectrumWorker(sampleRate float64) *spectrumWorker {
	w := &spectrumWorker{
		sampleRate: sampleRate,
		fft:        fourier.NewFFT(spectrumWindowSize),
	}
	for i := range w.window {
		value := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(spectrumWindowSize-1)))
		w.window[i] = value
		w.windowPower += value * value
	}

	upper := min(spectrumMaxHz, sampleRate/2)
	if upper > spectrumMinHz {
		ratio := upper / spectrumMinHz
		for i := range w.edges {
			w.edges[i] = spectrumMinHz * math.Pow(ratio, float64(i)/SpectrumBandCount)
		}
	}
	return w
}

// analyze applies the Hann window, transforms each channel independently, and
// averages per-bin stereo power. The normalization makes a full-scale sine
// approximately 0 dBFS after its spectral leakage is accumulated into a band.
func (w *spectrumWorker) analyze(samples [][2]float64) (bands [SpectrumBandCount]float64) {
	if len(samples) != spectrumWindowSize || w.sampleRate <= 0 || w.edges[SpectrumBandCount] <= spectrumMinHz {
		return bands
	}

	peak := 0.0
	for i, sample := range samples {
		peak = max(peak, math.Abs(sample[0]), math.Abs(sample[1]))
		w.left[i] = sample[0] * w.window[i]
		w.right[i] = sample[1] * w.window[i]
	}
	if peak < spectrumSilencePeak {
		return bands
	}

	w.fft.Coefficients(w.leftCoeff[:], w.left[:])
	w.fft.Coefficients(w.rightCoeff[:], w.right[:])
	baseScale := 2 / (spectrumWindowSize * w.windowPower)
	for bin := range w.powers {
		leftPower := abs2(w.leftCoeff[bin])
		rightPower := abs2(w.rightCoeff[bin])
		scale := baseScale
		if bin > 0 && bin < len(w.powers)-1 {
			scale *= 2 // Include the omitted negative-frequency coefficient.
		}
		w.powers[bin] = (leftPower + rightPower) * 0.5 * scale
	}

	binHz := w.sampleRate / spectrumWindowSize
	for band := range bands {
		power := w.bandPower(w.edges[band], w.edges[band+1], binHz)
		if power <= 0 {
			continue
		}
		db := 10 * math.Log10(power)
		bands[band] = min(max((db-spectrumFloorDB)/-spectrumFloorDB, 0), 1)
	}
	return bands
}

// bandPower assigns each FFT bin the width of its overlap with the requested
// frequency range, avoiding hard jumps when a logarithmic band edge cuts a bin.
func (w *spectrumWorker) bandPower(lowHz, highHz, binHz float64) float64 {
	if highHz <= lowHz || binHz <= 0 {
		return 0
	}
	first := max(1, int(math.Floor(lowHz/binHz-0.5)))
	last := min(len(w.powers)-1, int(math.Ceil(highHz/binHz+0.5)))
	power := 0.0
	for bin := first; bin <= last; bin++ {
		binLow := max(0, (float64(bin)-0.5)*binHz)
		binHigh := (float64(bin) + 0.5) * binHz
		overlap := min(highHz, binHigh) - max(lowHz, binLow)
		if overlap > 0 {
			power += w.powers[bin] * overlap / binHz
		}
	}
	return power
}

func abs2(value complex128) float64 {
	realPart := real(value)
	imagPart := imag(value)

	return realPart*realPart + imagPart*imagPart
}
