package player

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpicode/tmus/internal/app/archive"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/wav"
)

func decodeFile(path string) (beep.StreamSeekCloser, beep.Format, error) {
	if handler := archive.DefaultRegistry().FindHandler(path); handler != nil {
		return decodeArchiveEntry(handler, path)
	}
	ext := strings.ToLower(filepath.Ext(path))
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("open file: %w", err)
	}

	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
	)

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	case ".flac":
		streamer, format, err = flac.Decode(f)
	case ".opus", ".ogg", ".oga":
		streamer, format, err = decodeOgg(f)
	case ".m4a", ".mp4":
		streamer, format, err = decodeM4a(f)
	default:
		_ = f.Close()
		return nil, beep.Format{}, fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		_ = f.Close()
		return nil, beep.Format{}, fmt.Errorf("decode %s: %w", ext, err)
	}

	if streamer == nil {
		_ = f.Close()
		return nil, beep.Format{}, errors.New("decoder returned nil streamer")
	}

	return streamer, format, nil
}

func decodeArchiveEntry(handler archive.Handler, path string) (beep.StreamSeekCloser, beep.Format, error) {
	rc, err := handler.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("read archive entry: %w", err)
	}

	ext := filepath.Ext(path)
	if archive.IsArchivePath(path) {
		if inner := archive.EntryExt(path); inner != "" {
			ext = inner
		}
	}

	r := bytes.NewReader(data)
	return decodeReader(r, ext)
}

func decodeReader(r io.ReadSeeker, ext string) (beep.StreamSeekCloser, beep.Format, error) {
	ext = strings.ToLower(ext)
	rc := readSeekCloser{r: r}

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
		streamer, format, err = decodeOgg(rc)
	case ".m4a", ".mp4":
		streamer, format, err = decodeM4a(rc)
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

type readSeekCloser struct {
	r io.ReadSeeker
}

func (r readSeekCloser) Read(p []byte) (int, error)         { return r.r.Read(p) }
func (r readSeekCloser) Seek(o int64, w int) (int64, error) { return r.r.Seek(o, w) }
func (r readSeekCloser) Close() error                       { return nil }
