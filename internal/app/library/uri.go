package library

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func isRemote(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

func baseName(value string) string {
	if isArchivePath(value) {
		if _, archivePath, inner, err := splitArchiveURI(value); err == nil {
			if inner != "" {
				return path.Base(inner)
			}
			return filepath.Base(archivePath)
		}
	}
	if isRemote(value) {
		if parsed, err := url.Parse(value); err == nil && parsed.Path != "" {
			name := path.Base(parsed.Path)
			if name != "/" && name != "." {
				return name
			}
		}
		return value // Fallback for root streams like http://example.com/
	}
	return filepath.Base(value)
}
