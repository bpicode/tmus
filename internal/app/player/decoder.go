package player

import (
	"errors"
	"fmt"
	"io"

	"github.com/bpicode/tmus/internal/app/library"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/wav"
)

// decodeSource decodes a library.AudioSource into a beep audio stream.
// It takes ownership of AudioSource.Reader, closing it on error or delegating
// responsibility to the returned streamer on success.
//
// If AudioSource.Reader implements io.ReadSeekCloser (local files, buffered archive
// entries) the reader is passed directly to the format decoder, enabling
// seeking on the returned stream. Otherwise the reader is passed as-is without
// any buffering, and Seek on the returned stream will return an error - the
// expected behaviour for non-seekable sources such as HTTP streams.
func decodeSource(s library.AudioSource) (beep.StreamSeekCloser, beep.Format, error) {
	rc := s.Reader

	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
		err      error
	)

	switch s.Format {
	case library.FormatMP3:
		streamer, format, err = mp3.Decode(rc)
	case library.FormatWAV:
		streamer, format, err = wav.Decode(rc)
	case library.FormatFLAC:
		streamer, format, err = flac.Decode(rc)
	case library.FormatOPUS, library.FormatOGG, library.FormatOGA:
		rsc, ok := rc.(io.ReadSeekCloser)
		if !ok {
			_ = rc.Close()
			return nil, beep.Format{}, fmt.Errorf("ogg/opus decoding requires a seekable source")
		}
		streamer, format, err = decodeOgg(rsc)
	case library.FormatM4A, library.FormatMP4:
		rsc, ok := rc.(io.ReadSeekCloser)
		if !ok {
			_ = rc.Close()
			return nil, beep.Format{}, fmt.Errorf("m4a decoding requires a seekable source")
		}
		streamer, format, err = decodeM4a(rsc)
	default:
		_ = rc.Close()
		return nil, beep.Format{}, fmt.Errorf("unsupported file format: %v", s.Format)
	}

	if err != nil {
		_ = rc.Close()
		return nil, beep.Format{}, fmt.Errorf("decode format %v: %w", s.Format, err)
	}
	if streamer == nil {
		_ = rc.Close()
		return nil, beep.Format{}, errors.New("decoder returned nil streamer")
	}
	return streamer, format, nil
}
