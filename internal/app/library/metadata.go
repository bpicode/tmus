package library

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/bpicode/tmus/internal/app/archive"
	"github.com/dhowden/tag"
)

// Picture captures embedded album artwork data.
type Picture struct {
	MIMEType    string
	Type        string
	Description string
	Data        []byte
}

// Metadata captures audio tags for display.
type Metadata struct {
	Artist      string
	Title       string
	Album       string
	AlbumArtist string
	Composer    string
	Genre       string
	Year        int
	Comment     string
	Lyrics      string
	Duration    time.Duration
	Picture     *Picture
}

// TrackInfo is a minimal view of track fields used for metadata enrichment.
type TrackInfo struct {
	Path   string
	Name   string
	Artist string
	Title  string
	Album  string
}

// ReadMetadataBasic reads basic audio tags (artist/title/album).
func ReadMetadataBasic(path string) (Metadata, error) {
	meta, err := readMetadata(path)
	if err != nil {
		return Metadata{}, err
	}
	if meta.Artist == "" && meta.Title == "" && meta.Album == "" {
		return Metadata{}, errors.New("no metadata")
	}
	return Metadata{
		Artist:   meta.Artist,
		Title:    meta.Title,
		Album:    meta.Album,
		Duration: meta.Duration,
	}, nil
}

// ReadMetadataExtended reads all supported audio tags (including artwork).
func ReadMetadataExtended(path string) (Metadata, error) {
	return readMetadata(path)
}

func readMetadata(path string) (Metadata, error) {
	if handler := archive.DefaultRegistry().FindHandler(path); handler != nil {
		rc, err := handler.Open(path)
		if err != nil {
			return Metadata{}, fmt.Errorf("open archived file: %w", err)
		}
		defer rc.Close()
		return readMetadataFromArchived(rc)
	}
	f, err := os.Open(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return readMetadataFrom(f)
}

func readMetadataFrom(r io.ReadSeeker) (Metadata, error) {
	meta, err := tag.ReadFrom(r)
	if err != nil {
		return Metadata{}, err
	}
	result := Metadata{
		Artist:      meta.Artist(),
		Title:       meta.Title(),
		Album:       meta.Album(),
		AlbumArtist: meta.AlbumArtist(),
		Composer:    meta.Composer(),
		Genre:       meta.Genre(),
		Year:        meta.Year(),
		Comment:     meta.Comment(),
		Lyrics:      meta.Lyrics(),
	}

	if raw := meta.Raw(); raw != nil {
		if tlen, ok := raw["TLEN"]; ok {
			if tlenStr, ok := tlen.(string); ok {
				if ms, err := strconv.ParseInt(tlenStr, 10, 64); err == nil {
					result.Duration = time.Duration(ms) * time.Millisecond
				}
			}
		}
	}
	if pic := meta.Picture(); pic != nil {
		data := make([]byte, len(pic.Data))
		copy(data, pic.Data)
		result.Picture = &Picture{
			MIMEType:    pic.MIMEType,
			Type:        fmt.Sprint(pic.Type),
			Description: pic.Description,
			Data:        data,
		}
	}
	return result, nil
}

func readMetadataFromArchived(rc io.Reader) (Metadata, error) {
	bs, err := io.ReadAll(rc)
	if err != nil {
		return Metadata{}, err
	}
	return readMetadataFrom(bytes.NewReader(bs))
}
