package library

import (
	"errors"
	"io"
)

var (
	errNotArchive = errors.New("not an archive path")

	// ErrArchiveMemberTooLarge is returned when a decoded archive member exceeds
	// the configured per-member size limit.
	ErrArchiveMemberTooLarge = errors.New("archive member exceeds maximum decoded size")
)

func (l *Library) openArchiveEntry(path string) (io.ReadCloser, error) {
	handler := l.archive.findHandler(path)
	if handler == nil {
		return nil, errNotArchive
	}

	rc, err := handler.open(path)
	if err != nil {
		return nil, err
	}

	return newLimitedReadCloser(rc, l.opts.MaxArchiveMemberBytes), nil
}

func newLimitedReadCloser(rc io.ReadCloser, limit int64) io.ReadCloser {
	return &limitedReadCloser{
		ReadCloser: rc,
		remaining:  limit,
	}
}

type limitedReadCloser struct {
	io.ReadCloser
	remaining int64
}

func (r *limitedReadCloser) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		var one [1]byte
		n, err := r.ReadCloser.Read(one[:])
		if n > 0 {
			return 0, ErrArchiveMemberTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.ReadCloser.Read(p)
	r.remaining -= int64(n)
	return n, err
}
