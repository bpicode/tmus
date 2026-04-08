package library

import (
	"fmt"
	"path/filepath"
	"strings"
)

func splitArchivePath(scheme, value string) (archivePath, inner string, err error) {
	if trimmed, ok := strings.CutPrefix(value, "arch://"+scheme+":"); ok {
		parts := strings.SplitN(trimmed, "::", 2)
		archivePath = parts[0]
		if len(parts) == 2 {
			inner = strings.TrimPrefix(parts[1], "/")
		}
		if archivePath == "" {
			return "", "", fmt.Errorf("archive path missing")
		}
		return archivePath, inner, nil
	}

	archivePath = value
	inner = ""
	return archivePath, inner, nil
}

// IsArchivePath reports whether the value is an archive URI.
func IsArchivePath(value string) bool {
	return strings.HasPrefix(value, "arch://")
}

// SplitArchivePath parses an archive URI into scheme, archive path, and inner path.
func SplitArchivePath(value string) (scheme, archivePath, inner string, err error) {
	if !IsArchivePath(value) {
		return "", "", "", fmt.Errorf("not an archive path")
	}
	trimmed := strings.TrimPrefix(value, "arch://")
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", "", fmt.Errorf("invalid archive path")
	}
	scheme = parts[0]
	archivePath, inner, err = splitArchivePath(scheme, value)
	return scheme, archivePath, inner, err
}

// BuildArchivePath builds an archive URI from scheme, archive path, and inner path.
func BuildArchivePath(scheme, archivePath, inner string) string {
	archivePath = filepath.Clean(archivePath)
	if inner == "" {
		return "arch://" + scheme + ":" + archivePath
	}
	return "arch://" + scheme + ":" + archivePath + "::" + inner
}

// EntryExt returns the file extension for the entry portion of an archive URI.
func EntryExt(value string) string {
	if !IsArchivePath(value) {
		return ""
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "arch://"), "::", 2)
	if len(parts) != 2 {
		return ""
	}
	inner := strings.TrimPrefix(parts[1], "/")
	if inner == "" {
		return ""
	}
	return filepath.Ext(inner)
}
