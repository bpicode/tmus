package library

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
)

type archiveFile struct {
	lib  *Library
	path string
	name string
}

func (a archiveFile) Path() string {
	return a.path
}

func (a archiveFile) Name() string {
	return a.name
}

func (a archiveFile) entryType() entryType {
	return entryArchive
}

func (a archiveFile) Hidden() bool {
	return strings.HasPrefix(a.name, ".")
}

func (a archiveFile) Open(_ context.Context) (AudioSource, error) {
	return AudioSource{}, errNotAudio
}

func (a archiveFile) IsAudio() bool {
	return false
}

func (a archiveFile) Parent() string {
	return filepath.Dir(a.path)
}

func (a archiveFile) IsDir() bool {
	return false
}

func (a archiveFile) IsArchive() bool {
	return true
}

func (a archiveFile) FilesystemPath() (string, bool) {
	return a.path, true
}

func (a archiveFile) CanBrowse() bool {
	return true
}

func (a archiveFile) BrowsePath() (string, bool) {
	lib := a.lib
	if lib == nil {
		lib = defaultLibrary
	}
	handler := lib.archive.findHandler(a.path)
	if handler == nil {
		return "", false
	}
	return buildArchivePath(handler.scheme(), a.path, ""), true
}

type archiveEntry struct {
	lib   *Library
	path  string
	name  string
	isDir bool
}

func (a archiveEntry) Path() string {
	return a.path
}

func (a archiveEntry) Name() string {
	return a.name
}

func (a archiveEntry) entryType() entryType {
	if a.isDir {
		return entryDir
	}
	return entryTypeFromPath(a.path)
}

func (a archiveEntry) Hidden() bool {
	return strings.HasPrefix(a.name, ".")
}

func (a archiveEntry) Open(ctx context.Context) (AudioSource, error) {
	if !a.IsAudio() {
		return AudioSource{}, errNotAudio
	}
	if a.entryType() == entryStream {
		return a.openShortcut(ctx)
	}
	return a.openAudio()
}

func (a archiveEntry) openAudio() (AudioSource, error) {
	rc, err := a.library().openArchiveEntry(a.path)
	if err != nil {
		return AudioSource{}, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return AudioSource{}, fmt.Errorf("read archive entry: %w", err)
	}
	return AudioSource{Reader: nopSeekCloser{bytes.NewReader(data)}, Format: formatFromPath(a.path)}, nil
}

func (a archiveEntry) openShortcut(ctx context.Context) (AudioSource, error) {
	rc, err := a.library().openArchiveEntry(a.path)
	if err != nil {
		return AudioSource{}, err
	}
	defer rc.Close()

	var uri string
	switch strings.ToLower(entryExt(a.path)) {
	case ".url":
		uri, err = parseURLShortcut(rc)
	case ".stream":
		uri, err = parseStreamShortcut(rc)
	default:
		return AudioSource{}, errNotAudio
	}
	if err != nil {
		return AudioSource{}, err
	}
	return openRemoteAudio(ctx, uri)
}

func (a archiveEntry) IsAudio() bool {
	return !a.isDir && entryTypeFromPath(a.path) != entryOther
}

func (a archiveEntry) Parent() string {
	scheme, archivePath, inner, err := splitArchiveURI(a.path)
	if err != nil {
		return ""
	}
	parent := path.Dir(inner)
	if parent == "." {
		parent = ""
	}
	return buildArchivePath(scheme, archivePath, parent)
}

func (a archiveEntry) IsDir() bool {
	return a.isDir
}

func (a archiveEntry) IsArchive() bool {
	return false
}

func (a archiveEntry) FilesystemPath() (string, bool) {
	return "", false
}

func (a archiveEntry) CanBrowse() bool {
	return a.isDir
}

func (a archiveEntry) BrowsePath() (string, bool) {
	if !a.CanBrowse() {
		return "", false
	}
	return a.path, true
}

func (a archiveEntry) library() *Library {
	if a.lib != nil {
		return a.lib
	}
	return defaultLibrary
}

type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }
