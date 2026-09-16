package player

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpectrumWorker(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate int
		frequency  float64
		leftGain   float64
		rightGain  float64
		wantSilent bool
	}{
		{name: "bass at 44.1 kHz", sampleRate: 44100, frequency: 100, leftGain: 1, rightGain: 1},
		{name: "midrange at 48 kHz", sampleRate: 48000, frequency: 1000, leftGain: 1, rightGain: 1},
		{name: "treble at 44.1 kHz", sampleRate: 44100, frequency: 10000, leftGain: 1, rightGain: 1},
		{name: "one channel averages power", sampleRate: 48000, frequency: 1000, leftGain: 1},
		{name: "silence", sampleRate: 48000, wantSilent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := newSpectrumWorker(float64(tt.sampleRate))
			samples := sineWindow(tt.sampleRate, tt.frequency, tt.leftGain, tt.rightGain)
			bands := worker.analyze(samples[:])

			if tt.wantSilent {
				assert.Zero(t, bands)
				return
			}

			wantBand := frequencyBand(worker.edges, tt.frequency)
			gotBand := strongestBand(bands)
			require.Equalf(t, wantBand, gotBand, "unexpected strongest band for %.0f Hz; bands=%v", tt.frequency, bands)
			assert.GreaterOrEqual(t, bands[gotBand], 0.85)
			assert.LessOrEqual(t, bands[gotBand], 1.0)
			for i, level := range bands {
				assert.GreaterOrEqualf(t, level, 0.0, "band %d", i)
				assert.LessOrEqualf(t, level, 1.0, "band %d", i)
			}
		})
	}
}

func TestSpectrumWorkerDecibelMapping(t *testing.T) {
	worker := newSpectrumWorker(48000)
	full := sineWindow(48000, 1000, 1, 1)
	quiet := sineWindow(48000, 1000, 0.1, 0.1)
	fullBands := worker.analyze(full[:])
	quietBands := worker.analyze(quiet[:])
	band := strongestBand(fullBands)

	// A 0.1 amplitude is -20 dBFS, one third of the configured 60 dB span.
	assert.InDelta(t, 1.0/3, fullBands[band]-quietBands[band], 0.02)

	belowThreshold := sineWindow(48000, 1000, spectrumSilencePeak/2, spectrumSilencePeak/2)
	assert.Zero(t, worker.analyze(belowThreshold[:]))
}

func TestSpectrumWorkerCombinesStereoAsPower(t *testing.T) {
	worker := newSpectrumWorker(48000)
	identical := sineWindow(48000, 1000, 1, 1)
	leftOnly := sineWindow(48000, 1000, 1, 0)
	opposing := sineWindow(48000, 1000, 1, -1)
	identicalBands := worker.analyze(identical[:])
	leftOnlyBands := worker.analyze(leftOnly[:])
	opposingBands := worker.analyze(opposing[:])
	band := strongestBand(identicalBands)

	assert.InDelta(t, 3.0103/60, identicalBands[band]-leftOnlyBands[band], 0.01)
	assert.InDelta(t, identicalBands[band], opposingBands[band], 1e-12)
}

func TestSpectrumStreamerPreservesSamplesAndReturnValues(t *testing.T) {
	analyzer := newSpectrumAnalyzer(beep.SampleRate(48000))
	t.Cleanup(analyzer.close)
	analyzer.startPlayback(1)
	source := &testStreamer{
		samples: [][2]float64{{0.1, -0.1}, {0.2, -0.2}, {0.3, -0.3}},
		ok:      false,
		err:     errors.New("stream error"),
	}
	streamer := analyzer.wrapStreamer(source)
	output := make([][2]float64, 5)
	n, ok := streamer.Stream(output)

	require.Equal(t, len(source.samples), n)
	assert.Equal(t, source.ok, ok)
	assert.Equal(t, source.samples, output[:n])
	assert.ErrorIs(t, streamer.Err(), source.err)
}

func TestSpectrumAnalyzerWindowOverlapAcrossChunks(t *testing.T) {
	analyzer := newSpectrumAnalyzer(beep.SampleRate(48000))
	t.Cleanup(analyzer.close)
	analyzer.startPlayback(1)
	samples := sineWindow(48000, 1000, 1, 1)
	sequence := analyzer.latestSnapshot().Sequence

	for offset, chunkIndex := 0, 0; offset < len(samples); chunkIndex++ {
		sizes := [...]int{1, 17, 511, 29, 256, 777}
		size := min(sizes[chunkIndex%len(sizes)], len(samples)-offset)
		analyzer.captureSamples(samples[offset : offset+size])
		offset += size
	}
	first := waitForSpectrum(t, analyzer, sequence)
	assert.Equal(t, uint64(spectrumWindowSize), first.EndFrame)

	analyzer.captureSamples(samples[:spectrumHopSize-1])
	assert.Equal(t, first.Sequence, analyzer.latestSnapshot().Sequence)
	analyzer.captureSamples(samples[:1])
	second := waitForSpectrum(t, analyzer, first.Sequence)
	assert.Equal(t, uint64(spectrumWindowSize+spectrumHopSize), second.EndFrame)
}

func TestSpectrumAnalyzerDropsWorkWithoutBlocking(t *testing.T) {
	analyzer := &spectrumAnalyzer{
		freeBuffers: make(chan *spectrumWorkBuffer, 1),
		workQueue:   make(chan spectrumWork, 1),
	}
	analyzer.freeBuffers <- new(spectrumWorkBuffer)
	analyzer.playbackID.Store(1)
	analyzer.generation.Store(1)
	samples := sineWindow(48000, 1000, 1, 1)

	started := time.Now()
	analyzer.captureSamples(samples[:])
	analyzer.captureSamples(samples[:spectrumHopSize])
	assert.Less(t, time.Since(started), 100*time.Millisecond)
	assert.Positive(t, analyzer.droppedWindows.Load())
}

func TestSpectrumAnalyzerRejectsStaleWork(t *testing.T) {
	analyzer := newSpectrumAnalyzer(beep.SampleRate(48000))
	t.Cleanup(analyzer.close)
	analyzer.startPlayback(1)
	stale := spectrumWork{playbackID: 1, generation: analyzer.generation.Load()}
	analyzer.resetGeneration(1)
	reset := analyzer.latestSnapshot()
	var bands [SpectrumBandCount]float64
	bands[0] = 1
	analyzer.publishSpectrum(stale, bands)
	assert.Equal(t, reset, analyzer.latestSnapshot())
}

func TestSpectrumAnalyzerLifecycle(t *testing.T) {
	analyzer := newSpectrumAnalyzer(beep.SampleRate(48000))
	t.Cleanup(analyzer.close)
	samples := sineWindow(48000, 1000, 1, 1)

	analyzer.startPlayback(7)
	started := analyzer.latestSnapshot()
	assert.Equal(t, uint64(7), started.PlaybackID)
	assert.NotZero(t, started.Generation)
	analyzer.captureSamples(samples[:])
	analyzed := waitForSpectrum(t, analyzer, started.Sequence)
	assert.Equal(t, uint64(7), analyzed.PlaybackID)
	assert.Equal(t, started.Generation, analyzed.Generation)

	analyzer.resetGeneration(7)
	reset := analyzer.latestSnapshot()
	assert.Greater(t, reset.Generation, analyzed.Generation)
	assert.Zero(t, reset.Bands)

	analyzer.invalidatePlayback()
	invalid := analyzer.latestSnapshot()
	assert.Zero(t, invalid.PlaybackID)
	assert.Zero(t, invalid.Bands)
}

func TestSpectrumCaptureAllocations(t *testing.T) {
	analyzer := newSpectrumAnalyzer(beep.SampleRate(48000))
	t.Cleanup(analyzer.close)
	analyzer.startPlayback(1)
	samples := sineWindow(48000, 440, 1, 1)
	chunk := samples[:256]

	allocs := testing.AllocsPerRun(100, func() {
		analyzer.captureSamples(chunk)
	})
	assert.Zero(t, allocs)
}

func BenchmarkSpectrumCapture(b *testing.B) {
	analyzer := newSpectrumAnalyzer(beep.SampleRate(48000))
	b.Cleanup(analyzer.close)
	analyzer.startPlayback(1)
	samples := sineWindow(48000, 440, 1, 1)
	chunk := samples[:256]
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		analyzer.captureSamples(chunk)
	}
}

func BenchmarkSpectrumFFT(b *testing.B) {
	worker := newSpectrumWorker(48000)
	samples := sineWindow(48000, 1000, 1, 1)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		worker.analyze(samples[:])
	}
}

type testStreamer struct {
	samples [][2]float64
	ok      bool
	err     error
}

func (s *testStreamer) Stream(output [][2]float64) (int, bool) {
	return copy(output, s.samples), s.ok
}

func (s *testStreamer) Err() error {
	return s.err
}

func sineWindow(sampleRate int, frequency, leftGain, rightGain float64) [spectrumWindowSize][2]float64 {
	var samples [spectrumWindowSize][2]float64
	for i := range samples {
		value := math.Sin(2 * math.Pi * frequency * float64(i) / float64(sampleRate))
		samples[i] = [2]float64{value * leftGain, value * rightGain}
	}
	return samples
}

func frequencyBand(edges [SpectrumBandCount + 1]float64, frequency float64) int {
	for i := range SpectrumBandCount {
		if frequency >= edges[i] && frequency < edges[i+1] {
			return i
		}
	}
	return SpectrumBandCount - 1
}

func strongestBand(bands [SpectrumBandCount]float64) int {
	strongest := 0
	for i := 1; i < len(bands); i++ {
		if bands[i] > bands[strongest] {
			strongest = i
		}
	}
	return strongest
}

func waitForSpectrum(t *testing.T, analyzer *spectrumAnalyzer, afterSequence uint64) SpectrumSnapshot {
	t.Helper()
	var snapshot SpectrumSnapshot
	require.Eventually(t, func() bool {
		snapshot = analyzer.latestSnapshot()
		return snapshot.Sequence > afterSequence
	}, time.Second, time.Millisecond)
	return snapshot
}
