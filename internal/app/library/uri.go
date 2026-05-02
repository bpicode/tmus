package library

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// IsRemote reports whether the given URI points to a remote network resource (e.g., HTTP/HTTPS).
func IsRemote(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

// IsURI reports whether the given path is a URI rather than a standard local file path.
func IsURI(path string) bool {
	return strings.Contains(path, "://")
}

// BaseName safely extracts a user-friendly base name from a file path or URI.
func BaseName(value string) string {
	if IsArchivePath(value) {
		if _, archivePath, inner, err := SplitArchivePath(value); err == nil {
			if inner != "" {
				return path.Base(inner)
			}
			return filepath.Base(archivePath)
		}
	}
	if IsRemote(value) {
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
