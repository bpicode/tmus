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

func isArchivePath(value string) bool {
	return strings.HasPrefix(value, "arch://")
}

func splitArchiveURI(value string) (scheme, archivePath, inner string, err error) {
	if !isArchivePath(value) {
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

func buildArchivePath(scheme, archivePath, inner string) string {
	archivePath = filepath.Clean(archivePath)
	if inner == "" {
		return "arch://" + scheme + ":" + archivePath
	}
	return "arch://" + scheme + ":" + archivePath + "::" + inner
}

func entryExt(value string) string {
	if !isArchivePath(value) {
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
