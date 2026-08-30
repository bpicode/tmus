package player

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"
	"time"

	"github.com/abema/go-mp4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestM4AOpenInvalidFile(t *testing.T) {
	buf := bytes.NewReader([]byte("invalid m4a data"))
	_, err := openM4A(buf)
	assert.Error(t, err)
}

func TestBuildM4ASampleTable(t *testing.T) {
	t.Run("empty inputs", func(t *testing.T) {
		samples, err := buildM4ASampleTable(nil, nil, nil, nil, 1)
		assert.Nil(t, samples)
		assert.Error(t, err)
	})

	t.Run("multiple samples in one chunk", func(t *testing.T) {
		sampleSizes := []uint32{100, 150}
		chunkOffsets := []uint64{1000}
		stscEntries := []mp4.StscEntry{
			{FirstChunk: 1, SamplesPerChunk: 2, SampleDescriptionIndex: 1},
		}
		sttsEntries := []mp4.SttsEntry{
			{SampleCount: 2, SampleDelta: 1024},
		}

		samples, err := buildM4ASampleTable(sampleSizes, chunkOffsets, stscEntries, sttsEntries, 1)
		require.NoError(t, err)
		require.Len(t, samples, 2)

		assert.Equal(t, uint64(1000), samples[0].Offset)
		assert.Equal(t, uint32(100), samples[0].Size)
		assert.Equal(t, uint32(1024), samples[0].Duration)

		assert.Equal(t, uint64(1100), samples[1].Offset)
		assert.Equal(t, uint32(150), samples[1].Size)
		assert.Equal(t, uint32(1024), samples[1].Duration)
	})

	t.Run("variable durations", func(t *testing.T) {
		sampleSizes := []uint32{200, 250}
		chunkOffsets := []uint64{2000, 3000}
		stscEntries := []mp4.StscEntry{
			{FirstChunk: 1, SamplesPerChunk: 1, SampleDescriptionIndex: 1},
		}
		sttsEntries := []mp4.SttsEntry{
			{SampleCount: 1, SampleDelta: 4000},
			{SampleCount: 1, SampleDelta: 4096},
		}

		samples, err := buildM4ASampleTable(sampleSizes, chunkOffsets, stscEntries, sttsEntries, 1)
		require.NoError(t, err)
		require.Len(t, samples, 2)

		assert.Equal(t, uint64(2000), samples[0].Offset)
		assert.Equal(t, uint32(200), samples[0].Size)
		assert.Equal(t, uint32(4000), samples[0].Duration)

		assert.Equal(t, uint64(3000), samples[1].Offset)
		assert.Equal(t, uint32(250), samples[1].Size)
		assert.Equal(t, uint32(4096), samples[1].Duration)
	})
}

func TestM4ASampleSizes(t *testing.T) {
	tests := []struct {
		name    string
		stsz    *mp4.Stsz
		want    []uint32
		wantErr string
	}{
		{
			name: "variable sizes",
			stsz: &mp4.Stsz{SampleCount: 2, EntrySize: []uint32{100, 200}},
			want: []uint32{100, 200},
		},
		{
			name: "constant size",
			stsz: &mp4.Stsz{SampleSize: 100, SampleCount: 2},
			want: []uint32{100, 100},
		},
		{
			name:    "excessive sample count",
			stsz:    &mp4.Stsz{SampleSize: 1, SampleCount: maxM4ASampleCount + 1},
			wantErr: "exceeds limit",
		},
		{
			name:    "empty variable-size sample",
			stsz:    &mp4.Stsz{SampleCount: 1, EntrySize: []uint32{0}},
			wantErr: "sample 0 is empty",
		},
		{
			name:    "oversized variable-size sample",
			stsz:    &mp4.Stsz{SampleCount: 1, EntrySize: []uint32{maxM4ASampleSize + 1}},
			wantErr: "sample 0 is too large",
		},
		{
			name:    "oversized constant-size sample",
			stsz:    &mp4.Stsz{SampleSize: maxM4ASampleSize + 1, SampleCount: 1},
			wantErr: "samples are too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boxes := []*mp4.BoxInfoWithPayload{
				{
					Info:    mp4.BoxInfo{Type: mp4.BoxTypeStsz()},
					Payload: tt.stsz,
				},
			}
			got, err := m4aSampleSizes(boxes)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenM4AContainers(t *testing.T) {
	tests := []struct {
		name       string
		codec      m4aCodecType
		config     []byte
		sampleSize uint8
		delta      uint32
	}{
		{
			name:       "AAC",
			codec:      m4aCodecAAC,
			config:     []byte{0x12, 0x10},
			sampleSize: 16,
			delta:      1024,
		},
		{
			name:       "ALAC",
			codec:      m4aCodecALAC,
			config:     testALACConfig(24, 2, 44100),
			sampleSize: 24,
			delta:      4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := []byte{1, 2, 3}
			second := []byte{4, 5, 6, 7}
			data := testM4AContainer(tt.codec, tt.config, tt.delta, first, second)

			r, err := openM4A(bytes.NewReader(data))
			require.NoError(t, err)
			assert.Equal(t, tt.codec, r.Codec())
			assert.Equal(t, tt.config, r.CodecConfig())
			assert.Equal(t, uint32(44100), r.SampleRate())
			assert.Equal(t, uint8(2), r.Channels())
			assert.Equal(t, tt.sampleSize, r.SampleSize())
			assert.Equal(t, 2, r.SampleCount())
			assert.Equal(t, m4aUnitsToDuration(uint64(2*tt.delta), 44100), r.Duration())

			got, err := r.ReadSample(0)
			require.NoError(t, err)
			assert.Equal(t, first, got)
			got, err = r.ReadSample(1)
			require.NoError(t, err)
			assert.Equal(t, second, got)
		})
	}
}

func TestDecodeM4AFixtures(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "AAC", data: testM4AAACFixture},
		{name: "ALAC", data: testM4AALACFixture},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := testM4AFixture(t, tt.data)
			decoder, format, err := decodeM4a(src)
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, decoder.Close())
			})

			assert.Equal(t, 8000, int(format.SampleRate))
			assert.Equal(t, 2, format.NumChannels)
			require.Positive(t, decoder.Len())

			frames := make([][2]float64, 256)
			n, ok := decoder.Stream(frames)
			require.True(t, ok)
			require.Positive(t, n)
			require.NoError(t, decoder.Err())

			require.NoError(t, decoder.Seek(decoder.Len()/2))
			n, ok = decoder.Stream(frames)
			require.True(t, ok)
			require.Positive(t, n)
			require.NoError(t, decoder.Err())
		})
	}
}

func TestOpenM4ASkipsNonAudioSampleTables(t *testing.T) {
	first := []byte{1, 2, 3}
	second := []byte{4, 5, 6, 7}
	data := testM4AContainer(m4aCodecAAC, []byte{0x12, 0x10}, 1024, first, second)
	data = testM4APrependVideoTrack(data, len(first)+len(second))

	r, err := openM4A(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, m4aCodecAAC, r.Codec())
	assert.Equal(t, 2, r.SampleCount())
}

func TestM4AReaderMethods(t *testing.T) {
	data := []byte("0123456789ABCDEF0123456789ABCDEF")
	r := &m4aReader{
		reader:      bytes.NewReader(data),
		codec:       m4aCodecAAC,
		codecConfig: []byte{0x12, 0x10},
		sampleRate:  44100,
		channels:    2,
		sampleSize:  16,
		timescale:   44100,
		duration:    2 * time.Second,
		samples: []m4aSampleInfo{
			{Offset: 0, Size: 10, Duration: 44100},
			{Offset: 10, Size: 10, Duration: 44100},
		},
		starts: []uint64{0, 44100},
	}

	assert.Equal(t, m4aCodecAAC, r.Codec())
	assert.Equal(t, []byte{0x12, 0x10}, r.CodecConfig())
	assert.Equal(t, uint32(44100), r.SampleRate())
	assert.Equal(t, uint8(2), r.Channels())
	assert.Equal(t, uint8(16), r.SampleSize())
	assert.Equal(t, 2*time.Second, r.Duration())
	assert.Equal(t, 2, r.SampleCount())

	// Read sample
	sample0, err := r.ReadSample(0)
	require.NoError(t, err)
	assert.Equal(t, []byte("0123456789"), sample0)

	sample1, err := r.ReadSample(1)
	require.NoError(t, err)
	assert.Equal(t, []byte("ABCDEF0123"), sample1)

	_, err = r.ReadSample(-1)
	assert.ErrorContains(t, err, "invalid sample index -1")

	_, err = r.ReadSample(2)
	assert.ErrorContains(t, err, "invalid sample index 2")

	// Sample time
	assert.Equal(t, 0*time.Second, r.SampleTime(0))
	assert.Equal(t, 1*time.Second, r.SampleTime(1))
	assert.Equal(t, 2*time.Second, r.SampleTime(2))

	// Seek to time
	assert.Equal(t, 0, r.SeekToTime(0))
	assert.Equal(t, 0, r.SeekToTime(500*time.Millisecond))
	assert.Equal(t, 1, r.SeekToTime(1*time.Second))
	assert.Equal(t, 1, r.SeekToTime(1500*time.Millisecond))
	assert.Equal(t, 2, r.SeekToTime(3*time.Second))
}

func TestM4AReaderRejectsInvalidSampleSizes(t *testing.T) {
	tests := []struct {
		name    string
		size    uint32
		wantErr string
	}{
		{name: "empty", size: 0, wantErr: "sample 0 is empty"},
		{name: "too large", size: maxM4ASampleSize + 1, wantErr: "sample 0 is too large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &m4aReader{
				reader:  bytes.NewReader(nil),
				samples: []m4aSampleInfo{{Size: tt.size}},
			}
			_, err := r.ReadSample(0)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func testM4AContainer(codec m4aCodecType, config []byte, delta uint32, samples ...[]byte) []byte {
	mdatPayload := bytes.Join(samples, nil)
	mdat := testM4ABox("mdat", mdatPayload)
	sizes := make([]uint32, len(samples))
	for i, sample := range samples {
		sizes[i] = uint32(len(sample))
	}

	entryType := "mp4a"
	var codecBox []byte
	if codec == m4aCodecALAC {
		entryType = "alac"
		codecBox = testM4ABox("alac", config)
	} else {
		esdsPayload := append([]byte{0, 0, 0, 0, mp4.DecSpecificInfoTag, byte(len(config))}, config...)
		codecBox = testM4ABox("esds", esdsPayload)
	}
	entry := testM4AAudioEntry(entryType, codecBox)

	stsdBody := append(testM4AU32(1), entry...)
	stsd := testM4AFullBox("stsd", stsdBody)
	stts := testM4AFullBox("stts", append(testM4AU32(1), testM4AU32(uint32(len(samples)), delta)...))
	stsc := testM4AFullBox("stsc", append(testM4AU32(1), testM4AU32(1, uint32(len(samples)), 1)...))
	stszBody := append(testM4AU32(0, uint32(len(samples))), testM4AU32(sizes...)...)
	stsz := testM4AFullBox("stsz", stszBody)
	stco := testM4AFullBox("stco", append(testM4AU32(1), testM4AU32(8)...))
	stbl := testM4ABox("stbl", bytes.Join([][]byte{stsd, stts, stsc, stsz, stco}, nil))
	minf := testM4ABox("minf", stbl)

	mdhdBody := testM4AU32(0, 0, 44100, uint32(len(samples))*delta)
	mdhdBody = append(mdhdBody, 0, 0, 0, 0)
	mdhd := testM4AFullBox("mdhd", mdhdBody)
	hdlrBody := append(testM4AU32(0), []byte("soun")...)
	hdlrBody = append(hdlrBody, make([]byte, 13)...)
	hdlr := testM4AFullBox("hdlr", hdlrBody)
	mdia := testM4ABox("mdia", bytes.Join([][]byte{mdhd, hdlr, minf}, nil))
	moov := testM4ABox("moov", testM4ABox("trak", mdia))
	return append(mdat, moov...)
}

func testM4AAudioEntry(boxType string, child []byte) []byte {
	payload := make([]byte, 28)
	binary.BigEndian.PutUint16(payload[6:8], 1)
	binary.BigEndian.PutUint16(payload[16:18], 2)
	binary.BigEndian.PutUint16(payload[18:20], 16)
	binary.BigEndian.PutUint32(payload[24:28], 44100<<16)
	return testM4ABox(boxType, append(payload, child...))
}

func testM4APrependVideoTrack(data []byte, mediaSize int) []byte {
	mdatSize := 8 + mediaSize
	audioTrack := data[mdatSize+8:]

	mdhdBody := append(testM4AU32(0, 0, 1000, 1000), 0, 0, 0, 0)
	mdhd := testM4AFullBox("mdhd", mdhdBody)
	hdlrBody := append(testM4AU32(0), []byte("vide")...)
	hdlrBody = append(hdlrBody, make([]byte, 13)...)
	hdlr := testM4AFullBox("hdlr", hdlrBody)
	malformedStsz := testM4ABox("stsz", []byte{0})
	minf := testM4ABox("minf", testM4ABox("stbl", malformedStsz))
	videoTrack := testM4ABox("trak", testM4ABox("mdia", bytes.Join([][]byte{mdhd, hdlr, minf}, nil)))

	moovPayload := append(videoTrack, audioTrack...)
	return append(data[:mdatSize], testM4ABox("moov", moovPayload)...)
}

func testALACConfig(sampleSize, channels uint8, sampleRate uint32) []byte {
	config := make([]byte, 28)
	binary.BigEndian.PutUint32(config[4:8], 4096)
	config[9] = sampleSize
	config[13] = channels
	binary.BigEndian.PutUint32(config[24:28], sampleRate)
	return config
}

func testM4AFullBox(boxType string, body []byte) []byte {
	return testM4ABox(boxType, append([]byte{0, 0, 0, 0}, body...))
}

func testM4ABox(boxType string, payload []byte) []byte {
	box := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(box[:4], uint32(len(box)))
	copy(box[4:8], boxType)
	copy(box[8:], payload)
	return box
}

func testM4AU32(values ...uint32) []byte {
	data := make([]byte, 4*len(values))
	for i, value := range values {
		binary.BigEndian.PutUint32(data[i*4:], value)
	}
	return data
}

type testM4AReadSeekCloser struct {
	*bytes.Reader
}

func (*testM4AReadSeekCloser) Close() error {
	return nil
}

func testM4AFixture(t *testing.T, encoded string) *testM4AReadSeekCloser {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	return &testM4AReadSeekCloser{Reader: bytes.NewReader(data)}
}

// Generated with FFmpeg:
// ffmpeg -f lavfi -i anullsrc=r=8000:cl=mono -t 0.04 -map_metadata -1
// -fflags +bitexact -flags:a +bitexact -c:a aac -b:a 8k
// -movflags +faststart aac.m4a
const testM4AAACFixture = `AAAAHGZ0eXBNNEEgAAACAE00QSBpc29taXNvMgAAAtZtb292AAAAbG12aGQAAAAAAAAAAAAAAAAAAAPoAAAAKAABAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAACJXRyYWsAAABcdGtoZAAAAAMAAAAAAAAAAAAAAAEAAAAAAAAAKAAAAAAAAAAAAAAAAQEAAAAAAQAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAACRlZHRzAAAAHGVsc3QAAAAAAAAAAQAAACgAAAQAAAEAAAAAAZ1tZGlhAAAAIG1kaGQAAAAAAAAAAAAAAAAAAB9AAAAFQFXEAAAAAAAtaGRscgAAAAAAAAAAc291bgAAAAAAAAAAAAAAAFNvdW5kSGFuZGxlcgAAAAFIbWluZgAAABBzbWhkAAAAAAAAAAAAAAAkZGluZgAAABxkcmVmAAAAAAAAAAEAAAAMdXJsIAAAAAEAAAEMc3RibAAAAGpzdHNkAAAAAAAAAAEAAABabXA0YQAAAAAAAAABAAAAAAAAAAAAAQAQAAAAAB9AAAAAAAA2ZXNkcwAAAAADgICAJQABAASAgIAXQBUAAAAAAB9AAAABfAWAgIAFFYhW5QAGgICAAQIAAAAgc3R0cwAAAAAAAAACAAAAAQAABAAAAAABAAABQAAAABxzdHNjAAAAAAAAAAEAAAABAAAAAgAAAAEAAAAUc3RzegAAAAAAAAAEAAAAAgAAABRzdGNvAAAAAAAAAAEAAAMCAAAAGnNncGQBAAAAcm9sbAAAAAIAAAAB//8AAAAcc2JncAAAAAByb2xsAAAAAQAAAAIAAAABAAAAPXVkdGEAAAA1bWV0YQAAAAAAAAAhaGRscgAAAAAAAAAAbWRpcmFwcGwAAAAAAAAAAAAAAAAIaWxzdAAAAAhmcmVlAAAAEG1kYXQBGCAHARggBw==`

// Generated with FFmpeg:
// ffmpeg -f lavfi -i anullsrc=r=8000:cl=mono -t 0.04 -map_metadata -1
// -fflags +bitexact -flags:a +bitexact -c:a alac -movflags +faststart alac.m4a
const testM4AALACFixture = `AAAAHGZ0eXBNNEEgAAACAE00QSBpc29taXNvMgAAAoZtb292AAAAbG12aGQAAAAAAAAAAAAAAAAAAAPoAAAAKAABAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAAB1XRyYWsAAABcdGtoZAAAAAMAAAAAAAAAAAAAAAEAAAAAAAAAKAAAAAAAAAAAAAAAAQEAAAAAAQAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAACRlZHRzAAAAHGVsc3QAAAAAAAAAAQAAACgAAAAAAAEAAAAAAU1tZGlhAAAAIG1kaGQAAAAAAAAAAAAAAAAAAB9AAAABQFXEAAAAAAAtaGRscgAAAAAAAAAAc291bgAAAAAAAAAAAAAAAFNvdW5kSGFuZGxlcgAAAAD4bWluZgAAABBzbWhkAAAAAAAAAAAAAAAkZGluZgAAABxkcmVmAAAAAAAAAAEAAAAMdXJsIAAAAAEAAAC8c3RibAAAAFhzdHNkAAAAAAAAAAEAAABIYWxhYwAAAAAAAAABAAAAAAAAAAAAAQAQAAAAAB9AAAAAAAAkYWxhYwAAAAAAABAAABAoCg4BAAAAACAEAAH0AAAAH0AAAAAYc3R0cwAAAAAAAAABAAAAAQAAAUAAAAAcc3RzYwAAAAAAAAABAAAAAQAAAAEAAAABAAAAFHN0c3oAAAAAAAAAFwAAAAEAAAAUc3RjbwAAAAAAAAABAAACsgAAAD11ZHRhAAAANW1ldGEAAAAAAAAAIWhkbHIAAAAAAAAAAG1kaXJhcHBsAAAAAAAAAAAAAAAACGlsc3QAAAAIZnJlZQAAAB9tZGF0AAAQAAACgAAADwgBAAAAAAAAAP+An/A=`
