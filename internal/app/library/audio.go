package library

import (
	"path/filepath"
	"strings"
)

var supportedAudioExt = map[string]struct{}{
	".mp3":    {},
	".wav":    {},
	".flac":   {},
	".opus":   {},
	".ogg":    {},
	".oga":    {},
	".m4a":    {},
	".mp4":    {},
	".url":    {},
	".stream": {},
}

// IsAudio reports whether the path is a supported audio file (including archive entries).
func IsAudio(path string) bool {
	if IsRemote(path) {
		// Remote URLs are accepted optimistically; format validation defers to the decoder.
		return true
	}
	ext := filepath.Ext(path)
	if IsArchivePath(path) {
		if innerExt := EntryExt(path); innerExt != "" {
			ext = innerExt
		}
	}
	ext = strings.ToLower(ext)
	_, ok := supportedAudioExt[ext]
	return ok
}
