package lyrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseLrcLinePreservesTextAfterTimestamps(t *testing.T) {
	for _, text := range []string{"lyrics", "[annotation] lyrics", "[unclosed lyrics"} {
		t.Run(text, func(t *testing.T) {
			times, got, ok := parseLrcLine("[00:01][00:02]" + text)
			assert.True(t, ok)
			assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, times)
			assert.Equal(t, text, got)
		})
	}
}
