package library

import (
	"context"
	"io"
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
