package library

import (
	"path/filepath"
	"testing"

	_ "github.com/bpicode/tmus/testing"
	"github.com/stretchr/testify/assert"
)

func TestReadMetadataBasic(t *testing.T) {
	tests := []struct {
		name string
		file string
		want Metadata
	}{
		{
			name: "Britney Sheers",
			file: "Britney Sheers - Maybe One More Line.mp3",
			want: Metadata{
				Artist: "Britney Sheers",
				Title:  "Maybe One More Line",
			},
		},
		{
			name: "Metalguy-ca",
			file: "Metalguy-ca - Master of Carpets.mp3",
			want: Metadata{
				Artist: "Metalguy-ca",
				Title:  "Master of Carpets",
				Album:  "MoC Deluxe Edition",
			},
		},
		{
			name: "Nervana",
			file: "Nervana - Smells Like Cheap Spirit.mp3",
			want: Metadata{
				Artist: "Nervana",
				Title:  "Smells Like Cheap Spirit",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", tt.file)
			got, err := ReadMetadataBasic(path)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadMetadataBasicError(t *testing.T) {
	path := filepath.Join("testdata", "does-not-exist.mp3")
	_, err := ReadMetadataBasic(path)
	assert.Error(t, err)
}

func TestReadMetadataExtended(t *testing.T) {
	path := filepath.Join("testdata", "Metalguy-ca - Master of Carpets.mp3")
	m, err := ReadMetadataExtended(path)
	assert.NoError(t, err)
	assert.Equal(t, "Metalguy-ca", m.Artist)
	assert.Equal(t, "Master of Carpets", m.Title)
	assert.Equal(t, "MoC Deluxe Edition", m.Album)
	assert.NotNil(t, m.Picture)
	assert.NotEmpty(t, m.Lyrics)
}

func TestReadMetadataExtendedError(t *testing.T) {
	path := filepath.Join("testdata", "does-not-exist.mp3")
	_, err := ReadMetadataExtended(path)
	assert.Error(t, err)
}
