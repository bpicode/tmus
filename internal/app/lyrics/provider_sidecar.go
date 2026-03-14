package lyrics

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpicode/tmus/internal/app/archive"
)

// sidecarProvider resolves lyrics from .lrc or .txt files next to the track.
type sidecarProvider struct{}

// NewSidecarProvider creates a provider for sidecar lyric files.
func NewSidecarProvider() Provider {
	return sidecarProvider{}
}

// Name returns the provider source.
func (p sidecarProvider) name() Source {
	return "sidecar files"
}

// Find checks for sidecar lyric files in the same directory as the track.
func (p sidecarProvider) find(track TrackInfo) (Lyrics, error) {
	if track.Path == "" {
		return Lyrics{}, errors.New("track path is empty")
	}
	if archive.IsArchivePath(track.Path) {
		return Lyrics{}, errors.New("sidecar files are not supported in archives")
	}
	dir := filepath.Dir(track.Path)
	base := filepath.Base(track.Path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = base
	}
	candidates := []string{
		filepath.Join(dir, name+".lrc"),
		filepath.Join(dir, name+".txt"),
	}
	for _, path := range candidates {
		content, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Lyrics{}, err
		}
		raw := string(content)
		lines, timed := parse(raw)
		if len(lines) == 0 && strings.TrimSpace(raw) == "" {
			return Lyrics{}, errors.New("0 lines extracted from lyrics")
		}
		return Lyrics{
			Lines:      lines,
			Timed:      timed,
			Raw:        raw,
			Source:     p.name(),
			SourcePath: path,
		}, nil
	}
	return Lyrics{}, errors.New("no sidecar lyrics found")
}
