package library

import (
	"context"
	"strings"
)

type remoteEntry struct {
	path string
	name string
}

func (r remoteEntry) Path() string {
	return r.path
}

func (r remoteEntry) Name() string {
	return r.name
}

func (r remoteEntry) entryType() entryType {
	return entryStream
}

func (r remoteEntry) Hidden() bool {
	return strings.HasPrefix(r.name, ".")
}

func (r remoteEntry) Open(ctx context.Context) (AudioSource, error) {
	return openRemoteAudio(ctx, r.path)
}

func (r remoteEntry) IsAudio() bool {
	return true
}

func (r remoteEntry) Parent() string {
	return ""
}

func (r remoteEntry) IsDir() bool {
	return false
}

func (r remoteEntry) IsArchive() bool {
	return false
}

func (r remoteEntry) FilesystemPath() (string, bool) {
	return "", false
}

func (r remoteEntry) CanBrowse() bool {
	return false
}

func (r remoteEntry) BrowsePath() (string, bool) {
	return "", false
}
