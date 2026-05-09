package library

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// HTTPResolver handles http:// and https:// URIs, including m3u/pls playlists.
type HTTPResolver struct {
	Client *http.Client
}

// NewHTTPResolver creates a resolver for network streams.
// The client uses transport-level timeouts for connection setup and header
// reads without bounding the duration of long-lived audio streams.
func NewHTTPResolver() *HTTPResolver {
	return &HTTPResolver{
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
			},
		},
	}
}

// CanResolve returns true if the URI is a remote HTTP/HTTPS resource.
func (r *HTTPResolver) CanResolve(uri string) bool {
	return IsRemote(uri)
}

// Resolve opens the remote stream, resolving playlists (.m3u, .pls) and mapping content types to extensions.
func (r *HTTPResolver) Resolve(ctx context.Context, uri string) (Source, error) {
	return r.resolve(ctx, uri, 0)
}

func (r *HTTPResolver) resolve(ctx context.Context, uri string, depth int) (Source, error) {
	if depth > 5 {
		return Source{}, errors.New("playlist recursion limit exceeded")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return Source{}, err
	}

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
		targetURI, err := parsePlaylist(resp.Body, ext, uri)
		_ = resp.Body.Close()
		if err != nil {
			return Source{}, err
		}
		// Resolve the target recursively
		return r.resolve(ctx, targetURI, depth+1)
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

func parsePlaylist(r io.Reader, ext string, baseURI string) (string, error) {
	baseURL, _ := url.Parse(baseURI)
	isPls := ext == ".pls"

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var target string
		if isPls {
			if strings.HasPrefix(strings.ToLower(line), "file") {
				if _, after, found := strings.Cut(line, "="); found {
					target = strings.TrimSpace(after)
				}
			}
		} else if !strings.HasPrefix(line, "#") {
			target = line
		}

		if target == "" {
			continue
		}

		if baseURL != nil {
			if parsed, err := baseURL.Parse(target); err == nil {
				target = parsed.String()
			}
		}

		if IsRemote(target) {
			return target, nil
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
