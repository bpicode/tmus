package archive

import "io"

// Entry is a single file or directory within an archive.
type Entry struct {
	Name  string
	Path  string
	IsDir bool
}

// Handler manages a specific archive format (zip, tar, etc.).
type Handler interface {
	Scheme() string
	IsArchivePath(path string) bool
	List(path string, showHidden bool) ([]Entry, error)
	Open(path string) (io.ReadCloser, error)
}

// Registry is a collection of archive handlers.
type Registry struct {
	handlers []Handler
}

var defaultRegistry = func() *Registry {
	r := &Registry{}
	r.Register(NewZipHandler())
	r.Register(NewTarHandler())
	r.Register(NewTarGzHandler())
	r.Register(NewTarXzHandler())
	r.Register(NewSevenZipHandler())
	r.Register(NewRarHandler())
	return r
}()

// DefaultRegistry returns the shared registry of built-in handlers.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register adds a handler. Later handlers are checked after earlier ones.
func (r *Registry) Register(handler Handler) {
	if handler == nil {
		return
	}
	r.handlers = append(r.handlers, handler)
}

// FindHandler returns a handler that recognizes the path.
func (r *Registry) FindHandler(path string) Handler {
	for _, h := range r.handlers {
		if h.IsArchivePath(path) {
			return h
		}
	}
	return nil
}
