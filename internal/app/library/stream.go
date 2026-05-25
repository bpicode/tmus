package library

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ParseStreamFile parses a .url or .stream file and returns the destination URL.
// For .url files, it looks for the URL key under [InternetShortcut].
// For .stream files, it returns the first non-empty, non-comment line.
func ParseStreamFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".url":
		return ParseURLShortcut(file)
	case ".stream":
		return ParseStreamShortcut(file)
	default:
		return "", os.ErrNotExist
	}
}

// ParseURLShortcut parses a Windows Internet Shortcut stream and returns its URL.
func ParseURLShortcut(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	inSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.ToLower(line)
			if section == "[internetshortcut]" {
				inSection = true
			} else {
				inSection = false
			}
			continue
		}
		if inSection || !strings.Contains(line, "[") {
			if strings.HasPrefix(strings.ToLower(line), "url=") {
				return strings.TrimSpace(line[4:]), nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}

// ParseStreamShortcut parses a .stream file and returns its first stream URL.
func ParseStreamShortcut(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		return line, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}

// ResolvePlayable converts a user-provided file path or remote URI into a resolved path
// and a clean display name. If the input is not a supported audio or stream shortcut source,
// it returns false as the third value.
func ResolvePlayable(value string) (resolvedURL string, cleanName string, ok bool) {
	if value == "" {
		return "", "", false
	}
	path := value
	if !IsURI(value) {
		path = filepath.Clean(value)
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	if !IsAudio(path) {
		return "", "", false
	}
	return resolveStreamShortcut(path)
}

func resolveStreamShortcut(path string) (resolvedURL string, cleanName string, ok bool) {
	if IsRemote(path) {
		return path, BaseName(path), true
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".url" || ext == ".stream" {
		resolved, err := ParseStreamFile(path)
		if err != nil || resolved == "" {
			return "", "", false
		}
		name := BaseName(path)
		name = strings.TrimSuffix(name, filepath.Ext(path))
		return resolved, name, true
	}
	return path, BaseName(path), true
}
