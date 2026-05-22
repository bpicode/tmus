package library

import (
	"io"
)

type archiveHandler interface {
	scheme() string
	isArchivePath(path string) bool
	list(path string, showHidden bool) ([]Entry, error)
	open(path string) (io.ReadCloser, error)
}

type archiveRegistry struct {
	handlers []archiveHandler
}

var defaultArchiveRegistry = func() *archiveRegistry {
	r := &archiveRegistry{}
	r.register(newZipHandler())
	r.register(newTarHandler())
	r.register(newTarGzHandler())
	r.register(newTarXzHandler())
	r.register(newSevenZipHandler())
	r.register(newRarHandler())
	return r
}()

func archiveHandlers() *archiveRegistry {
	return defaultArchiveRegistry
}

func (r *archiveRegistry) register(handler archiveHandler) {
	if handler == nil {
		return
	}
	r.handlers = append(r.handlers, handler)
}

func (r *archiveRegistry) findHandler(path string) archiveHandler {
	for _, h := range r.handlers {
		if h.isArchivePath(path) {
			return h
		}
	}
	return nil
}
