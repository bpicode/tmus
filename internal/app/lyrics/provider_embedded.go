package lyrics

import "errors"

// RawLyricsReader returns raw embedded lyrics for a track path.
type RawLyricsReader func(path string) (string, error)

// embeddedProvider resolves lyrics from embedded tags using the provided reader.
type embeddedProvider struct {
	read RawLyricsReader
}

// NewEmbeddedProvider creates a provider for embedded tags.
func NewEmbeddedProvider(read RawLyricsReader) Provider {
	return embeddedProvider{read: read}
}

// Name returns the provider source.
func (p embeddedProvider) name() Source {
	return "embedded tags"
}

// Find reads embedded lyrics using the configured reader.
func (p embeddedProvider) find(track TrackInfo) (Lyrics, error) {
	if track.Path == "" {
		return Lyrics{}, errors.New("track path is empty")
	}
	if p.read == nil {
		return Lyrics{}, errors.New("no reader for track")
	}
	raw, err := p.read(track.Path)
	if err != nil {
		return Lyrics{}, err
	}
	lines, timed := parse(raw)
	if len(lines) == 0 {
		return Lyrics{}, errors.New("0 lines extracted from lyrics")
	}
	return Lyrics{
		Lines:      lines,
		Timed:      timed,
		Raw:        raw,
		Source:     p.name(),
		SourcePath: track.Path,
	}, nil
}
