package library

import (
	"io"
)

// ArchiveEntry is a single file or directory within an archive.
type ArchiveEntry struct {
	Name  string
	Path  string
	IsDir bool
}

// ArchiveHandler manages a specific archive format (zip, tar, etc.).
type ArchiveHandler interface {
	Scheme() string
	IsArchivePath(path string) bool
	List(path string, showHidden bool) ([]ArchiveEntry, error)
	Open(path string) (io.ReadCloser, error)
}

// ArchiveRegistry is a collection of archive handlers.
type ArchiveRegistry struct {
	handlers []ArchiveHandler
}

var defaultArchiveRegistry = func() *ArchiveRegistry {
	r := &ArchiveRegistry{}
	r.Register(NewZipHandler())
	r.Register(NewTarHandler())
	r.Register(NewTarGzHandler())
	r.Register(NewTarXzHandler())
	r.Register(NewSevenZipHandler())
	r.Register(NewRarHandler())
	return r
}()

// DefaultArchiveRegistry returns the shared registry of built-in handlers.
func DefaultArchiveRegistry() *ArchiveRegistry {
	return defaultArchiveRegistry
}

// Register adds a handler. Later handlers are checked after earlier ones.
func (r *ArchiveRegistry) Register(handler ArchiveHandler) {
	if handler == nil {
		return
	}
	r.handlers = append(r.handlers, handler)
}

// FindHandler returns a handler that recognizes the path.
func (r *ArchiveRegistry) FindHandler(path string) ArchiveHandler {
	for _, h := range r.handlers {
		if h.IsArchivePath(path) {
			return h
		}
	}
	return nil
}
