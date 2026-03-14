package lyrics

import (
	"testing"

	"github.com/bpicode/tmus/internal/app/library"
	_ "github.com/bpicode/tmus/testing"
	"github.com/stretchr/testify/assert"
)

func TestResolver_Find(t *testing.T) {
	tests := []struct {
		name     string
		resolver *Resolver
		track    TrackInfo
		contains []Line
	}{
		{
			name: "embedded ID3 tags",
			resolver: NewResolver(NewEmbeddedProvider(func(path string) (string, error) {
				m, err := library.ReadMetadataExtended(path)
				return m.Lyrics, err
			})),
			track: TrackInfo{
				Path: "testdata/Metalguy-ca - Master of Carpets.mp3",
			},
			contains: []Line{
				{
					Text:    "End of breakfast play, crumbs are blown away",
					Time:    0,
					HasTime: true,
				},
				{
					Text:    "I'm your source of floor-wide suction!",
					Time:    3000000000,
					HasTime: true,
				},
			},
		},
		{
			name:     "sidecar lrc file",
			resolver: NewResolver(NewSidecarProvider()),
			track: TrackInfo{
				Path: "testdata/Britney Sheers - Maybe One More Line.mp3",
			},
			contains: []Line{
				{
					Text:    "When I’m not high I lose my mind",
					Time:    14000000000,
					HasTime: true,
				},
				{
					Text:    "Give me a tray",
					Time:    17000000000,
					HasTime: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := tt.resolver.Find(tt.track)
			assert.NoError(t, err)
			for _, expectedLine := range tt.contains {
				assert.Contains(t, l.Lines, expectedLine)
			}
		})
	}
}
