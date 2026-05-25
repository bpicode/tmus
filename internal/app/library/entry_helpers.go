package library

import (
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
	default:
		return EntryOther
	}
}

func formatFromPath(path string) FormatType {
	switch entryTypeFromPath(path) {
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
