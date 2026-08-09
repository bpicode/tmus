package library

import (
	"fmt"
	"path/filepath"
	"strings"
)

func splitArchivePath(scheme, value string) (archivePath, inner string, err error) {
	if trimmed, ok := strings.CutPrefix(value, "arch://"+scheme+":"); ok {
		archivePath, inner, _ = strings.Cut(trimmed, "::")
		inner = strings.TrimPrefix(inner, "/")
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
	trimmed, _ := strings.CutPrefix(value, "arch://")
	scheme, _, ok := strings.Cut(trimmed, ":")
	if !ok || scheme == "" {
		return "", "", "", fmt.Errorf("invalid archive path")
	}
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
	trimmed, _ := strings.CutPrefix(value, "arch://")
	_, inner, ok := strings.Cut(trimmed, "::")
	if !ok {
		return ""
	}
	inner = strings.TrimPrefix(inner, "/")
	if inner == "" {
		return ""
	}
	return filepath.Ext(inner)
}
