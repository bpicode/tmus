package library

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// HTTPResolver handles http:// and https:// URIs, including m3u/pls playlists.
type HTTPResolver struct {
	Client *http.Client
}

// NewHTTPResolver creates a resolver for network streams.
func NewHTTPResolver() *HTTPResolver {
	return &HTTPResolver{
		Client: http.DefaultClient,
	}
}

// CanResolve returns true if the URI is a remote HTTP/HTTPS resource.
func (r *HTTPResolver) CanResolve(uri string) bool {
	return IsRemote(uri)
}

// Resolve opens the remote stream, resolving playlists (.m3u, .pls) and mapping content types to extensions.
func (r *HTTPResolver) Resolve(ctx context.Context, uri string) (Source, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return Source{}, err
	}
	// We might need to pretend to be a standard browser or player, some radios block Go-http-client
	req.Header.Set("User-Agent", "tmus/1.0")

	resp, err := r.Client.Do(req)
	if err != nil {
		return Source{}, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return Source{}, fmt.Errorf("http error: %s", resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	ext := extFromContentType(contentType)
	if ext == "" {
		if u, err := url.Parse(uri); err == nil {
			ext = filepath.Ext(u.Path)
		} else {
			ext = filepath.Ext(uri)
		}
	}
	ext = strings.ToLower(ext)

	if isPlaylist(contentType, ext) {
		defer resp.Body.Close()
		targetURI, err := parsePlaylist(resp.Body, ext)
		if err != nil {
			return Source{}, err
		}
		// Resolve the target recursively
		return r.Resolve(ctx, targetURI)
	}

	return Source{
		Reader: resp.Body,
		Ext:    ext,
	}, nil
}

func isPlaylist(contentType, ext string) bool {
	if strings.Contains(contentType, "mpegurl") || strings.Contains(contentType, "scpls") {
		return true
	}
	return ext == ".m3u" || ext == ".m3u8" || ext == ".pls"
}

func parsePlaylist(r io.Reader, ext string) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if ext == ".pls" || strings.Contains(ext, "scpls") {
			if strings.HasPrefix(strings.ToLower(line), "file") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		} else { // m3u / generic
			if !strings.HasPrefix(line, "#") && IsRemote(line) {
				return line, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no valid stream URL found in playlist")
}

func extFromContentType(ct string) string {
	ct = strings.ToLower(ct)
	if strings.Contains(ct, "audio/mpeg") {
		return ".mp3"
	}
	if strings.Contains(ct, "audio/ogg") || strings.Contains(ct, "application/ogg") {
		return ".ogg"
	}
	if strings.Contains(ct, "audio/aac") {
		return ".m4a" // Let m4a/aac decode it
	}
	if strings.Contains(ct, "audio/wav") {
		return ".wav"
	}
	if strings.Contains(ct, "audio/flac") {
		return ".flac"
	}
	return ""
}
