package library

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Source is the result of resolving a URI to a raw audio byte stream.
// Reader may additionally implement io.ReadSeeker for seekable access.
type Source struct {
	// Reader provides the raw audio content. The caller is responsible for
	// closing it. It may implement io.ReadSeeker if seeking is supported by
	// the underlying source (all local sources guarantee this).
	Reader io.ReadCloser
	// Ext is the lowercase file-extension format hint (e.g. ".mp3").
	// May be empty when the format cannot be determined from the URI alone.
	Ext string
}

// SourceResolver resolves a URI to a raw audio byte stream.
// It is deliberately codec-agnostic; callers handle audio decoding.
// Implementations must be safe for concurrent use.
type SourceResolver interface {
	// CanResolve reports whether this resolver handles the given URI.
	CanResolve(uri string) bool
	// Resolve opens the URI and returns a raw audio content stream.
	// The caller is responsible for closing Source.Reader.
	Resolve(ctx context.Context, uri string) (Source, error)
}

// LocalResolver handles local filesystem paths and archive URIs (arch://...).
type LocalResolver struct{}

// CanResolve returns true for any URI that is not an HTTP or HTTPS URL.
func (LocalResolver) CanResolve(uri string) bool {
	return !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://")
}

// Resolve opens a local file or archive entry.
// The returned Source.Reader also implements io.ReadSeeker for all local sources.
func (LocalResolver) Resolve(_ context.Context, uri string) (Source, error) {
	if handler := DefaultArchiveRegistry().FindHandler(uri); handler != nil {
		return resolveArchiveEntry(handler, uri)
	}
	f, err := os.Open(uri)
	if err != nil {
		return Source{}, fmt.Errorf("open file: %w", err)
	}
	return Source{Reader: f, Ext: strings.ToLower(filepath.Ext(uri))}, nil
}

// resolveArchiveEntry buffers the archive entry so it can be decoded
// as a seekable byte stream. The returned Reader also implements io.ReadSeeker.
func resolveArchiveEntry(handler ArchiveHandler, uri string) (Source, error) {
	rc, err := handler.Open(uri)
	if err != nil {
		return Source{}, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return Source{}, fmt.Errorf("read archive entry: %w", err)
	}

	ext := filepath.Ext(uri)
	if IsArchivePath(uri) {
		if inner := EntryExt(uri); inner != "" {
			ext = inner
		}
	}
	return Source{Reader: nopSeekCloser{bytes.NewReader(data)}, Ext: strings.ToLower(ext)}, nil
}

// nopSeekCloser wraps an io.ReadSeeker with a no-op Close.
type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }
