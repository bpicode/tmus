package player

import (
	"testing"

	_ "github.com/bpicode/tmus/testing"
	"github.com/gopxl/beep/v2"
	"github.com/stretchr/testify/assert"
)

func Test_decodeFile(t *testing.T) {
	tests := []struct {
		arg        string
		wantFormat beep.Format
	}{
		{
			arg:        "testdata/Britney Sheers - Maybe One More Line.mp3",
			wantFormat: beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2},
		},
		{
			arg:        "testdata/Metalguy-ca - Master of Carpets.mp3",
			wantFormat: beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2},
		},
		{
			arg:        "testdata/Nervana - Smells Like Cheap Spirit.mp3",
			wantFormat: beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			ssc, f, err := decodeFile(tt.arg)
			assert.NoError(t, err)
			defer ssc.Close()
			assert.NotNil(t, ssc)
			assert.Equal(t, tt.wantFormat, f)
		})
	}
}
