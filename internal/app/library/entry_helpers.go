package library

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
)

var errNotAudio = errors.New("not an audio file")

func entryTypeFromPath(path string) entryType {
	ext := filepath.Ext(path)
	if isArchivePath(path) {
		if innerExt := entryExt(path); innerExt != "" {
			ext = innerExt
		}
	}
	return entryTypeFromExt(ext)
}

func entryTypeFromExt(ext string) entryType {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3":
		return entryMP3
	case ".flac":
		return entryFLAC
	case ".ogg":
		return entryOGG
	case ".opus":
		return entryOPUS
	case ".oga":
		return entryOGA
	case ".m4a":
		return entryM4A
	case ".mp4":
		return entryMP4
	case ".wav":
		return entryWAV
	case ".url", ".stream":
		return entryStream
	default:
		return entryOther
	}
}

func formatFromPath(path string) FormatType {
	ext := filepath.Ext(path)
	if isArchivePath(path) {
		if innerExt := entryExt(path); innerExt != "" {
			ext = innerExt
		}
	}
	return formatFromExt(ext)
}

func formatFromExt(ext string) FormatType {
	switch entryTypeFromExt(ext) {
	case entryMP3:
		return FormatMP3
	case entryFLAC:
		return FormatFLAC
	case entryOGG:
		return FormatOGG
	case entryOPUS:
		return FormatOPUS
	case entryOGA:
		return FormatOGA
	case entryM4A:
		return FormatM4A
	case entryMP4:
		return FormatMP4
	case entryWAV:
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
	return newHTTPResolver().resolve(ctx, uri)
}
