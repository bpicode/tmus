package library

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewICYReaderRejectsInvalidInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
	}{
		{name: "zero", interval: 0},
		{name: "negative", interval: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := newICYReader(io.NopCloser(strings.NewReader("")), tt.interval, "")

			assert.Nil(t, reader)
			assert.Error(t, err)
		})
	}
}

func TestNewICYReaderIgnoresOversizedStationName(t *testing.T) {
	reader, err := newICYReader(
		io.NopCloser(strings.NewReader("")),
		1,
		strings.Repeat("a", maxICYStationNameBytes+1),
	)
	require.NoError(t, err)

	assert.Empty(t, reader.station)
	select {
	case metadata := <-reader.updates:
		t.Fatalf("unexpected initial metadata: %#v", metadata)
	default:
	}
	assert.NoError(t, reader.Close())
}

func TestICYReaderStripsMetadataAndPublishesSnapshot(t *testing.T) {
	stream := append([]byte("abcd"), icyMetadataBlock("StreamTitle='Artist - Track';")...)
	stream = append(stream, []byte("efgh")...)
	stream = append(stream, 0) // No metadata after the second audio interval.

	reader, err := newICYReader(io.NopCloser(bytes.NewReader(stream)), 4, " Station ")
	require.NoError(t, err)

	audio, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "abcdefgh", string(audio))

	metadata, ok := <-reader.updates
	require.True(t, ok)
	assert.Equal(t, SourceMetadata{
		DisplayTitle: "Artist - Track",
		Artist:       "Artist",
		Title:        "Track",
		SourceName:   "Station",
	}, metadata)
	_, ok = <-reader.updates
	assert.False(t, ok)
	assert.NoError(t, reader.Close())
}

func TestParseICYMetadata(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   SourceMetadata
		wantOK bool
	}{
		{
			name:   "artist and title",
			raw:    "StreamTitle='Artist - Track';",
			want:   SourceMetadata{DisplayTitle: "Artist - Track", Artist: "Artist", Title: "Track", SourceName: "Station"},
			wantOK: true,
		},
		{
			name:   "title only",
			raw:    "StreamTitle='Track';",
			want:   SourceMetadata{DisplayTitle: "Track", Title: "Track", SourceName: "Station"},
			wantOK: true,
		},
		{
			name:   "empty title clears snapshot",
			raw:    "StreamTitle='';",
			want:   SourceMetadata{SourceName: "Station"},
			wantOK: true,
		},
		{
			name: "missing title",
			raw:  "StreamUrl='https://example.com';",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := icyMetadataBlock(tt.raw)
			got, ok := parseICYMetadata(block[1:], "Station")

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestICYReaderReportsFramingErrors(t *testing.T) {
	tests := []struct {
		name        string
		stream      []byte
		wantError   error
		wantContext string
	}{
		{
			name:        "missing metadata length",
			stream:      []byte("abcd"),
			wantError:   io.EOF,
			wantContext: "read ICY metadata length",
		},
		{
			name:        "truncated metadata block",
			stream:      append([]byte("abcd\x01"), []byte("short")...),
			wantError:   io.ErrUnexpectedEOF,
			wantContext: "read ICY metadata block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := newICYReader(io.NopCloser(bytes.NewReader(tt.stream)), 4, "")
			require.NoError(t, err)

			audio, err := io.ReadAll(reader)
			assert.Equal(t, "abcd", string(audio))
			assert.ErrorIs(t, err, tt.wantError)
			assert.ErrorContains(t, err, tt.wantContext)
			_, ok := <-reader.updates
			assert.False(t, ok)
			assert.NoError(t, reader.Close())
		})
	}
}

func icyMetadataBlock(metadata string) []byte {
	blocks := (len(metadata) + 15) / 16
	block := make([]byte, 1+blocks*16)
	block[0] = byte(blocks)
	copy(block[1:], metadata)
	return block
}
