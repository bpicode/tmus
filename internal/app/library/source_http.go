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

type httpResolver struct {
	client *http.Client
}

// The client uses transport-level timeouts for connection setup and header
// reads without bounding the duration of long-lived audio streams.
func newHTTPResolver() *httpResolver {
	return &httpResolver{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
			},
		},
	}
}

func (r *httpResolver) resolve(ctx context.Context, uri string) (AudioSource, error) {
	return r.resolvePlaylist(ctx, uri, 0)
}

func (r *httpResolver) resolvePlaylist(ctx context.Context, uri string, depth int) (AudioSource, error) {
	if depth > 5 {
		return AudioSource{}, errors.New("playlist recursion limit exceeded")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return AudioSource{}, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return AudioSource{}, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return AudioSource{}, fmt.Errorf("http error: %s", resp.Status)
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
			return AudioSource{}, err
		}
		return r.resolvePlaylist(ctx, targetURI, depth+1)
	}

	return AudioSource{Reader: resp.Body, Format: formatFromExt(ext)}, nil
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

		if isRemote(target) {
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
	if strings.Contains(ct, "audio/wav") {
		return ".wav"
	}
	if strings.Contains(ct, "audio/flac") {
		return ".flac"
	}
	return ""
}
