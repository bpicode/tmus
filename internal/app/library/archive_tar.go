package library

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/ulikunitz/xz"
)

type tarHandler struct {
	scheme string
	exts   []string
	gzip   bool
	xz     bool
}

func NewTarHandler() ArchiveHandler {
	return &tarHandler{scheme: "tar", exts: []string{".tar"}}
}

func NewTarGzHandler() ArchiveHandler {
	return &tarHandler{scheme: "targz", exts: []string{".tar.gz", ".tgz"}, gzip: true}
}

func NewTarXzHandler() ArchiveHandler {
	return &tarHandler{scheme: "tarxz", exts: []string{".tar.xz", ".txz"}, xz: true}
}

func (h *tarHandler) Scheme() string {
	return h.scheme
}

func (h *tarHandler) IsArchivePath(value string) bool {
	prefix := "arch://" + h.scheme + ":"
	if strings.HasPrefix(value, prefix) {
		return true
	}
	lower := strings.ToLower(value)
	for _, ext := range h.exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func (h *tarHandler) List(value string, showHidden bool) ([]Entry, error) {
	archivePath, inner, err := splitArchivePath(h.scheme, value)
	if err != nil {
		return nil, err
	}

	reader, err := openTarReader(archivePath, h.gzip, h.xz)
	if err != nil {
		return nil, err
	}
	defer reader.close()

	if inner != "" && !strings.HasSuffix(inner, "/") {
		inner += "/"
	}

	children := map[string]Entry{}
	for {
		hdr, err := reader.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		name := hdr.Name
		if inner != "" {
			if !strings.HasPrefix(name, inner) {
				continue
			}
			name = strings.TrimPrefix(name, inner)
		}
		if name == "" {
			continue
		}
		parts := strings.Split(name, "/")
		child := parts[0]
		if child == "" {
			continue
		}
		if !showHidden && strings.HasPrefix(child, ".") {
			continue
		}
		entryPath := path.Join(inner, child)
		if len(parts) > 1 || hdr.FileInfo().IsDir() {
			path := BuildArchivePath(h.scheme, archivePath, strings.TrimSuffix(entryPath, "/"))
			children[child] = Entry{
				Name:  child,
				Path:  path,
				IsDir: true,
			}
		} else {
			path := BuildArchivePath(h.scheme, archivePath, entryPath)
			children[child] = Entry{
				Name:    child,
				Path:    path,
				IsAudio: IsAudio(path),
			}
		}
	}

	entries := make([]Entry, 0, len(children))
	for _, v := range children {
		entries = append(entries, v)
	}
	return entries, nil
}

func (h *tarHandler) Open(value string) (io.ReadCloser, error) {
	archivePath, inner, err := splitArchivePath(h.scheme, value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return nil, fmt.Errorf("tar path missing entry")
	}

	reader, err := openTarReader(archivePath, h.gzip, h.xz)
	if err != nil {
		return nil, err
	}

	for {
		hdr, err := reader.next()
		if err == io.EOF {
			reader.close()
			return nil, fmt.Errorf("tar entry not found: %s", inner)
		}
		if err != nil {
			reader.close()
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Name != inner {
			continue
		}
		if hdr.FileInfo().IsDir() {
			reader.close()
			return nil, fmt.Errorf("tar entry is a directory: %s", inner)
		}
		return reader.readCloser(hdr.Size), nil
	}
}

type tarReader struct {
	file *os.File
	gz   *gzip.Reader
	xz   *xz.Reader
	tr   *tar.Reader
}

func openTarReader(path string, gz bool, useXZ bool) (*tarReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tar: %w", err)
	}
	var r io.Reader = f
	var gzReader *gzip.Reader
	var xzReader *xz.Reader
	if gz {
		gzReader, err = gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("open gzip: %w", err)
		}
		r = gzReader
	}
	if useXZ {
		xzReader, err = xz.NewReader(r)
		if err != nil {
			if gzReader != nil {
				_ = gzReader.Close()
			}
			_ = f.Close()
			return nil, fmt.Errorf("open xz: %w", err)
		}
		r = xzReader
	}
	return &tarReader{file: f, gz: gzReader, xz: xzReader, tr: tar.NewReader(r)}, nil
}

func (t *tarReader) next() (*tar.Header, error) {
	return t.tr.Next()
}

func (t *tarReader) readCloser(size int64) io.ReadCloser {
	return &tarEntryReader{r: io.LimitReader(t.tr, size), closeFn: t.close}
}

func (t *tarReader) close() error {
	if t.gz != nil {
		_ = t.gz.Close()
	}
	if t.file != nil {
		return t.file.Close()
	}
	return nil
}

type tarEntryReader struct {
	r       io.Reader
	closeFn func() error
}

func (t *tarEntryReader) Read(p []byte) (int, error) {
	return t.r.Read(p)
}

func (t *tarEntryReader) Close() error {
	if t.closeFn != nil {
		return t.closeFn()
	}
	return nil
}
