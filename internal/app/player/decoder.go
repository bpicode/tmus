package player

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bpicode/tmus/internal/app/library"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/wav"
)

// decodeSource decodes a library.Source into a beep audio stream.
// It takes ownership of Source.Reader, closing it on error or delegating
// responsibility to the returned streamer on success.
//
// If Source.Reader implements io.ReadSeekCloser (local files, buffered archive
// entries) the reader is passed directly to the format decoder, enabling
// seeking on the returned stream. Otherwise the reader is passed as-is without
// any buffering, and Seek on the returned stream will return an error — the
// expected behaviour for non-seekable sources such as HTTP streams.
func decodeSource(s library.Source) (beep.StreamSeekCloser, beep.Format, error) {
	rc := s.Reader
	ext := strings.ToLower(s.Ext)

	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
		err      error
	)

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(rc)
	case ".wav":
		streamer, format, err = wav.Decode(rc)
	case ".flac":
		streamer, format, err = flac.Decode(rc)
	case ".opus", ".ogg", ".oga":
		rsc, ok := rc.(io.ReadSeekCloser)
		if !ok {
			_ = rc.Close()
			return nil, beep.Format{}, fmt.Errorf("ogg/opus decoding requires a seekable source")
		}
		streamer, format, err = decodeOgg(rsc)
	case ".m4a", ".mp4":
		rsc, ok := rc.(io.ReadSeekCloser)
		if !ok {
			_ = rc.Close()
			return nil, beep.Format{}, fmt.Errorf("m4a decoding requires a seekable source")
		}
		streamer, format, err = decodeM4a(rsc)
	default:
		_ = rc.Close()
		return nil, beep.Format{}, fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		_ = rc.Close()
		return nil, beep.Format{}, fmt.Errorf("decode %s: %w", ext, err)
	}
	if streamer == nil {
		_ = rc.Close()
		return nil, beep.Format{}, errors.New("decoder returned nil streamer")
	}
	return streamer, format, nil
}
