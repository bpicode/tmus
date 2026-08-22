package library

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const maxICYStationNameBytes = 4 << 10

type icyReader struct {
	rc        io.ReadCloser
	interval  int
	remaining int
	station   string
	updates   chan SourceMetadata
	last      SourceMetadata
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

func newICYReader(rc io.ReadCloser, interval int, station string) (*icyReader, error) {
	if interval <= 0 {
		return nil, errors.New("metadata interval must be positive")
	}
	if len(station) > maxICYStationNameBytes {
		station = ""
	}
	station = strings.TrimSpace(decodeICYText([]byte(station)))
	r := &icyReader{
		rc:        rc,
		interval:  interval,
		remaining: interval,
		station:   station,
		updates:   make(chan SourceMetadata, 1),
	}
	if station != "" {
		r.publish(SourceMetadata{SourceName: station})
	}
	return r, nil
}

func (r *icyReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.remaining == 0 {
		if err := r.readMetadata(); err != nil {
			r.closeUpdates()
			return 0, err
		}
		r.remaining = r.interval
	}

	want := min(len(p), r.remaining)
	n, err := r.rc.Read(p[:want])
	r.remaining -= n
	if err != nil {
		r.closeUpdates()
	}
	return n, err
}

func (r *icyReader) Close() error {
	r.closeUpdates()
	return r.rc.Close()
}

func (r *icyReader) readMetadata() error {
	var size [1]byte
	if _, err := io.ReadFull(r.rc, size[:]); err != nil {
		return fmt.Errorf("read ICY metadata length: %w", err)
	}
	length := int(size[0]) * 16
	if length == 0 {
		return nil
	}

	raw := make([]byte, length)
	if _, err := io.ReadFull(r.rc, raw); err != nil {
		return fmt.Errorf("read ICY metadata block: %w", err)
	}
	metadata, ok := parseICYMetadata(raw, r.station)
	if ok {
		r.publish(metadata)
	}
	return nil
}

func (r *icyReader) publish(metadata SourceMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	if metadata == r.last {
		return
	}
	r.last = metadata
	select {
	case r.updates <- metadata:
		return
	default:
	}
	select {
	case <-r.updates:
	default:
	}
	select {
	case r.updates <- metadata:
	default:
	}
}

func (r *icyReader) closeUpdates() {
	r.closeOnce.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.closed = true
		close(r.updates)
	})
}

func parseICYMetadata(raw []byte, station string) (SourceMetadata, bool) {
	raw = bytes.TrimRight(raw, "\x00")
	text := decodeICYText(raw)
	rawTitle, hasTitle := icyField(text, "StreamTitle")
	if !hasTitle {
		return SourceMetadata{}, false
	}

	metadata := SourceMetadata{
		DisplayTitle: strings.TrimSpace(rawTitle),
		SourceName:   strings.TrimSpace(station),
	}
	if artist, title, found := strings.Cut(metadata.DisplayTitle, " - "); found {
		metadata.Artist = strings.TrimSpace(artist)
		metadata.Title = strings.TrimSpace(title)
	} else {
		metadata.Title = metadata.DisplayTitle
	}
	return metadata, true
}

func icyField(metadata, name string) (string, bool) {
	for pos := 0; pos < len(metadata); {
		for pos < len(metadata) && (isICYSpace(metadata[pos]) || metadata[pos] == ';') {
			pos++
		}

		fieldStart := pos
		for pos < len(metadata) && !isICYSpace(metadata[pos]) && metadata[pos] != '=' && metadata[pos] != ';' {
			pos++
		}
		field := metadata[fieldStart:pos]

		for pos < len(metadata) && isICYSpace(metadata[pos]) {
			pos++
		}
		if field == "" || pos >= len(metadata) || metadata[pos] != '=' {
			pos = nextICYField(metadata, pos)
			continue
		}
		pos++
		for pos < len(metadata) && isICYSpace(metadata[pos]) {
			pos++
		}
		if pos >= len(metadata) || metadata[pos] != '\'' {
			pos = nextICYField(metadata, pos)
			continue
		}
		pos++

		end := strings.Index(metadata[pos:], "';")
		if end < 0 {
			return "", false
		}
		if strings.EqualFold(field, name) {
			return metadata[pos : pos+end], true
		}
		pos += end + 2
	}
	return "", false
}

func isICYSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

func nextICYField(metadata string, pos int) int {
	next := strings.IndexByte(metadata[pos:], ';')
	if next < 0 {
		return len(metadata)
	}
	return pos + next + 1
}

func decodeICYText(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = charmap.Windows1252.DecodeByte(b)
	}
	return string(runes)
}
