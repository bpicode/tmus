package library

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
)

var errNotAudio = errors.New("not an audio file")

func entryTypeFromPath(path string) EntryType {
	ext := filepath.Ext(path)
	if IsArchivePath(path) {
		if innerExt := EntryExt(path); innerExt != "" {
			ext = innerExt
		}
	}
	return entryTypeFromExt(ext)
}

func entryTypeFromExt(ext string) EntryType {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3":
		return EntryMP3
	case ".flac":
		return EntryFLAC
	case ".ogg":
		return EntryOGG
	case ".opus":
		return EntryOPUS
	case ".oga":
		return EntryOGA
	case ".m4a":
		return EntryM4A
	case ".mp4":
		return EntryMP4
	case ".wav":
		return EntryWAV
	case ".url", ".stream":
		return EntryStream
	default:
		return EntryOther
	}
}

func formatFromPath(path string) FormatType {
	ext := filepath.Ext(path)
	if IsArchivePath(path) {
		if innerExt := EntryExt(path); innerExt != "" {
			ext = innerExt
		}
	}
	return formatFromExt(ext)
}

func formatFromExt(ext string) FormatType {
	switch entryTypeFromExt(ext) {
	case EntryMP3:
		return FormatMP3
	case EntryFLAC:
		return FormatFLAC
	case EntryOGG:
		return FormatOGG
	case EntryOPUS:
		return FormatOPUS
	case EntryOGA:
		return FormatOGA
	case EntryM4A:
		return FormatM4A
	case EntryMP4:
		return FormatMP4
	case EntryWAV:
		return FormatWAV
	default:
		return FormatUnknown
	}
}

func isURLFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".url")
}

func isStreamFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".stream")
}

func openRemoteAudio(ctx context.Context, uri string) (AudioSource, error) {
	source, err := NewHTTPResolver().Resolve(ctx, uri)
	if err != nil {
		return AudioSource{}, err
	}
	return AudioSource{Reader: source.Reader, Format: formatFromExt(source.Ext)}, nil
}
