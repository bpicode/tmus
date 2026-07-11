package config

import (
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByteSizeUnmarshalTextUnits(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ByteSize
	}{
		{name: "bytes", raw: "7B", want: 7},
		{name: "kilobytes", raw: "2KB", want: 2 * 1000},
		{name: "megabytes", raw: "512MB", want: 512 * 1000 * 1000},
		{name: "gigabytes", raw: "3GB", want: 3 * 1000 * 1000 * 1000},
		{name: "terabytes", raw: "1TB", want: 1 * 1000 * 1000 * 1000 * 1000},
		{name: "kibibytes", raw: "2KiB", want: 2 * 1024},
		{name: "mebibytes", raw: "512MiB", want: 512 * 1024 * 1024},
		{name: "gibibytes", raw: "3GiB", want: 3 * 1024 * 1024 * 1024},
		{name: "tebibytes", raw: "1TiB", want: 1 * 1024 * 1024 * 1024 * 1024},
		{name: "max int64 bytes", raw: "9223372036854775807B", want: 1<<63 - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ByteSize

			err := got.UnmarshalText([]byte(tt.raw))
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestByteSizeUnmarshalTextRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{"512M", "512mb", "512 MiB", "", "9223372036854775808B", "9223372036854776KB"} {
		t.Run(raw, func(t *testing.T) {
			var got ByteSize

			err := got.UnmarshalText([]byte(raw))

			assert.Error(t, err)
		})
	}
}

func TestByteSizeMarshalText(t *testing.T) {
	tests := []struct {
		name  string
		value ByteSize
		want  string
	}{
		{name: "bytes", value: 7, want: "7B"},
		{name: "mebibytes", value: 512 * 1024 * 1024, want: "512MiB"},
		{name: "megabytes", value: 512 * 1000 * 1000, want: "512MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.MarshalText()
			require.NoError(t, err)

			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestByteSizeTOMLStringUnits(t *testing.T) {
	var got struct {
		Size ByteSize `toml:"size"`
	}

	err := toml.Unmarshal([]byte(`size = "512MiB"`), &got)
	require.NoError(t, err)

	assert.Equal(t, ByteSize(512*1024*1024), got.Size)
}

func TestByteSizeTOMLBareNumberMeansBytes(t *testing.T) {
	var got struct {
		Size ByteSize `toml:"size"`
	}

	err := toml.Unmarshal([]byte("size = 512\n"), &got)
	require.NoError(t, err)

	assert.Equal(t, ByteSize(512), got.Size)
}

func TestByteSizeTOMLMarshal(t *testing.T) {
	value := struct {
		Size ByteSize `toml:"size"`
	}{
		Size: 512 * 1024 * 1024,
	}

	got, err := toml.Marshal(value)
	require.NoError(t, err)

	assert.Contains(t, string(got), `size = '512MiB'`)
}
