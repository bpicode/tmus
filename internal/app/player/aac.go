package player

import (
	"bufio"
	"context"
	"errors"
	"io"

	"github.com/gopxl/beep/v2"
	"github.com/llehouerou/go-faad2"
)

var errADTSNotSeekable = errors.New("aac: raw ADTS source is not seekable")

const (
	aacSamplesPerBlock = 1024
	aacProbeFrames     = 3
	aacMaxFrameSize    = 8191
)

// decodeAAC decodes raw AAC audio carried in an ADTS stream.
func decodeAAC(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	br := bufio.NewReaderSize(rc, aacProbeFrames*aacMaxFrameSize)
	sampleRate, channels, err := probeAACFormat(br)
	if err != nil {
		return nil, beep.Format{}, err
	}

	r, err := faad2.OpenADTS(context.Background(), br)
	if err != nil {
		return nil, beep.Format{}, err
	}

	format := beep.Format{
		SampleRate:  beep.SampleRate(sampleRate),
		NumChannels: 2,
		Precision:   2,
	}
	return &aacDecoder{
		reader:   r,
		closer:   rc,
		channels: channels,
	}, format, nil
}

// probeAACFormat detects the decoder's output channel count and SBR upsampling
// by inspecting PCM from the first useful ADTS frame. Peeking leaves the stream
// untouched for the playback decoder.
func probeAACFormat(r *bufio.Reader) (uint32, int, error) {
	header, err := r.Peek(7)
	if err != nil {
		return 0, 0, err
	}
	coreRate, headerChannels, _, err := faad2.ParseADTSHeader(header)
	if err != nil {
		return 0, 0, err
	}
	if headerChannels == 0 {
		return 0, 0, errors.New("aac: stream has no audio channels")
	}

	frequencyIndex := (header[2] >> 2) & 0x0f
	objectType := ((header[2] >> 6) & 0x03) + 1
	config := []byte{
		(objectType << 3) | ((frequencyIndex & 0x0e) >> 1),
		((frequencyIndex & 0x01) << 7) | (headerChannels << 3),
	}
	decoder, err := faad2.NewDecoder(context.Background())
	if err != nil {
		return 0, 0, err
	}
	defer decoder.Close(context.Background())
	if err := decoder.Init(context.Background(), config); err != nil {
		return 0, 0, err
	}
	outputChannels := int(decoder.Channels())
	if outputChannels == 0 {
		return 0, 0, errors.New("aac: decoder has no output channels")
	}

	offset := 0
	for range aacProbeFrames {
		frameHeader, err := r.Peek(offset + 7)
		if err != nil {
			return 0, 0, err
		}
		_, frameChannels, frameLength, err := faad2.ParseADTSHeader(frameHeader[offset:])
		if err != nil {
			return 0, 0, err
		}
		if frameChannels != headerChannels {
			return 0, 0, errors.New("aac: channel configuration changed during probe")
		}

		headerSize := 7
		if frameHeader[offset+1]&0x01 == 0 {
			headerSize = 9
		}
		if int(frameLength) <= headerSize {
			return 0, 0, faad2.ErrInvalidADTS
		}
		end := offset + int(frameLength)
		frame, err := r.Peek(end)
		if err != nil {
			return 0, 0, err
		}
		pcm, err := decoder.Decode(context.Background(), frame[offset+headerSize:end])
		if err != nil {
			return 0, 0, err
		}
		if len(pcm) > 0 {
			blocks := int(frame[offset+6]&0x03) + 1
			return decodedAACSampleRate(coreRate, outputChannels, blocks, len(pcm)), outputChannels, nil
		}
		offset = end
	}

	return coreRate, outputChannels, nil
}

func decodedAACSampleRate(coreRate uint32, channels, blocks, pcmSamples int) uint32 {
	if channels <= 0 || blocks <= 0 || pcmSamples%channels != 0 {
		return coreRate
	}
	coreFrames := aacSamplesPerBlock * blocks
	decodedFrames := pcmSamples / channels
	if decodedFrames < coreFrames || decodedFrames%coreFrames != 0 {
		return coreRate
	}
	return coreRate * uint32(decodedFrames/coreFrames)
}

type aacDecoder struct {
	reader   *faad2.ADTSReader
	closer   io.Closer
	channels int
	pcm      []int16
	position int
	err      error
	eof      bool
}

func (d *aacDecoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil || d.eof {
		return 0, false
	}
	if len(samples) == 0 {
		return 0, true
	}

	sampleCount := len(samples) * d.channels
	if cap(d.pcm) < sampleCount {
		d.pcm = make([]int16, sampleCount)
	} else {
		d.pcm = d.pcm[:sampleCount]
	}

	read, err := d.reader.Read(context.Background(), d.pcm)
	if err != nil {
		if errors.Is(err, io.EOF) {
			d.eof = true
		} else {
			d.err = err
		}
	}

	n = read / d.channels
	for i := range n {
		left := float64(d.pcm[i*d.channels]) / 32768.0
		right := left
		if d.channels > 1 {
			right = float64(d.pcm[i*d.channels+1]) / 32768.0
		}
		samples[i] = [2]float64{left, right}
	}
	d.position += n
	return n, n > 0
}

func (d *aacDecoder) Err() error {
	return d.err
}

func (d *aacDecoder) Len() int {
	return 0
}

func (d *aacDecoder) Position() int {
	return d.position
}

func (d *aacDecoder) Seek(int) error {
	return errADTSNotSeekable
}

func (d *aacDecoder) Close() error {
	decoderErr := d.reader.Close(context.Background())
	return errors.Join(decoderErr, d.closer.Close())
}
