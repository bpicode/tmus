package player

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io"
	"slices"
	"testing"

	"github.com/bpicode/tmus/internal/app/library"
	"github.com/gopxl/beep/v2"
	"github.com/llehouerou/go-faad2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAACBase64 contains two ADTS-framed AAC-LC silence frames generated with:
// ffmpeg -f lavfi -i anullsrc=r=8000:cl=mono -t 0.08 -c:a aac -b:a 16k -f adts test.aac
const testAACBase64 = "//FsQAOf/N4CAExhdmM2MC4zMS4xMDIAAjBADv/xbEABf/wBGCAH"

func TestDecodeAACStream(t *testing.T) {
	streamer, format, err := decodeSource(library.AudioSource{
		Reader: io.NopCloser(bytes.NewReader(testAACData(t))),
		Format: library.FormatAAC,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, streamer.Close())
	})

	// FAAD2 applies implicit SBR upsampling to this low-rate sample.
	assert.Equal(t, beep.Format{SampleRate: 16000, NumChannels: 2, Precision: 2}, format)
	assert.Zero(t, streamer.Len())
	assert.ErrorIs(t, streamer.Seek(0), errADTSNotSeekable)

	samples := make([][2]float64, 2048)
	n, ok := streamer.Stream(samples)
	assert.True(t, ok)
	assert.Positive(t, n)
	assert.Equal(t, n, streamer.Position())
}

func TestDecodedAACSampleRate(t *testing.T) {
	tests := []struct {
		name       string
		coreRate   uint32
		channels   int
		blocks     int
		pcmSamples int
		want       uint32
	}{
		{name: "AAC-LC stereo", coreRate: 44100, channels: 2, blocks: 1, pcmSamples: 2048, want: 44100},
		{name: "HE-AAC stereo", coreRate: 22050, channels: 2, blocks: 1, pcmSamples: 4096, want: 44100},
		{name: "HE-AAC mono", coreRate: 24000, channels: 1, blocks: 1, pcmSamples: 2048, want: 48000},
		{name: "multiple raw blocks", coreRate: 22050, channels: 2, blocks: 2, pcmSamples: 8192, want: 44100},
		{name: "incomplete PCM frame", coreRate: 44100, channels: 2, blocks: 1, pcmSamples: 2047, want: 44100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, decodedAACSampleRate(tt.coreRate, tt.channels, tt.blocks, tt.pcmSamples))
		})
	}
}

func TestProbeAACFormatRejectsMalformedInput(t *testing.T) {
	valid := testAACData(t)

	invalidSync := slices.Clone(valid)
	invalidSync[0] = 0

	shortFrame := slices.Clone(valid[:7])
	setADTSFrameLength(shortFrame, 7)

	_, _, firstFrameLength, err := faad2.ParseADTSHeader(valid)
	require.NoError(t, err)
	truncatedFrame := slices.Clone(valid[:firstFrameLength-1])

	changedChannels := slices.Clone(valid)
	setADTSChannels(changedChannels[firstFrameLength:], 2)

	tests := []struct {
		name        string
		data        []byte
		wantIs      error
		wantMessage string
	}{
		{name: "short header", data: valid[:6], wantIs: io.EOF},
		{name: "invalid sync word", data: invalidSync, wantIs: faad2.ErrADTSSyncNotFound},
		{name: "frame shorter than header", data: shortFrame, wantIs: faad2.ErrInvalidADTS},
		{name: "truncated frame", data: truncatedFrame, wantIs: io.EOF},
		{name: "channel configuration changes", data: changedChannels, wantMessage: "channel configuration changed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := probeAACFormat(bufio.NewReader(bytes.NewReader(tt.data)))
			require.Error(t, err)
			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			}
			if tt.wantMessage != "" {
				assert.ErrorContains(t, err, tt.wantMessage)
			}
		})
	}
}

func TestDecodeAACStreamWithCRCHeaders(t *testing.T) {
	streamer, format, err := decodeSource(library.AudioSource{
		Reader: io.NopCloser(bytes.NewReader(addADTSCRC(t, testAACData(t)))),
		Format: library.FormatAAC,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, streamer.Close())
	})

	assert.Equal(t, beep.SampleRate(16000), format.SampleRate)
	samples := make([][2]float64, 2048)
	n, ok := streamer.Stream(samples)
	assert.True(t, ok)
	assert.Positive(t, n)
}

func testAACData(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(testAACBase64)
	require.NoError(t, err)
	return data
}

func setADTSFrameLength(header []byte, frameLength uint16) {
	header[3] = (header[3] & 0xfc) | byte(frameLength>>11)
	header[4] = byte(frameLength >> 3)
	header[5] = (header[5] & 0x1f) | byte(frameLength&0x07)<<5
}

func setADTSChannels(header []byte, channels uint8) {
	header[2] = (header[2] & 0xfe) | (channels >> 2)
	header[3] = (header[3] & 0x3f) | (channels&0x03)<<6
}

func addADTSCRC(t *testing.T, data []byte) []byte {
	t.Helper()
	var result []byte
	for len(data) > 0 {
		require.GreaterOrEqual(t, len(data), 7)
		_, _, frameLength, err := faad2.ParseADTSHeader(data)
		require.NoError(t, err)
		require.LessOrEqual(t, int(frameLength), len(data))

		header := slices.Clone(data[:7])
		header[1] &^= 0x01
		setADTSFrameLength(header, frameLength+2)
		result = append(result, header...)
		result = append(result, 0, 0)
		result = append(result, data[7:frameLength]...)
		data = data[frameLength:]
	}
	return result
}

func FuzzProbeAACFormatDoesNotPanic(f *testing.F) {
	data, err := base64.StdEncoding.DecodeString(testAACBase64)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Add([]byte("not ADTS"))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > aacProbeFrames*aacMaxFrameSize {
			t.Skip()
		}
		_, _, _ = probeAACFormat(bufio.NewReader(bytes.NewReader(data)))
	})
}
