package library

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/dhowden/tag"
)

// Picture captures embedded artwork from an audio file.
type Picture struct {
	MIMEType    string
	Type        string
	Description string
	Data        []byte
}

// Metadata captures audio tags and related display data.
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

// TrackInfo is the track identity used for metadata enrichment.
type TrackInfo struct {
	Path   string
	Name   string
	Artist string
	Title  string
	Album  string
}

// ReadMetadataBasic reads artist, title, album, and duration metadata.
func ReadMetadataBasic(path string) (Metadata, error) {
	return defaultLibrary.ReadMetadataBasic(path)
}

// ReadMetadataBasic reads artist, title, album, and duration metadata.
func (l *Library) ReadMetadataBasic(path string) (Metadata, error) {
	meta, err := l.readMetadata(path)
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

// ReadMetadataExtended reads all supported metadata, including artwork.
func ReadMetadataExtended(path string) (Metadata, error) {
	return defaultLibrary.ReadMetadataExtended(path)
}

// ReadMetadataExtended reads all supported metadata, including artwork.
func (l *Library) ReadMetadataExtended(path string) (Metadata, error) {
	return l.readMetadata(path)
}

func (l *Library) readMetadata(path string) (Metadata, error) {
	if isRemote(path) {
		return Metadata{}, errors.New("remote metadata not supported")
	}
	if handler := l.archive.findHandler(path); handler != nil {
		rc, err := handler.open(path)
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
