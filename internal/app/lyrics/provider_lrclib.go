package lyrics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bpicode/tmus/internal/config"
)

type lrcLibProvider struct {
	enabled      bool
	cacheEnabled bool
	cacheDir     string
	httpClient   *http.Client
}

// NewLrcLibProvider creates a provider for the lrclib.net API.
func NewLrcLibProvider(cfg config.LrcLibConfig, cacheBaseDir string) Provider {
	return &lrcLibProvider{
		enabled:      cfg.Enabled,
		cacheEnabled: cfg.Cache.Enabled,
		cacheDir:     filepath.Join(cacheBaseDir, "lyrics", "lrclib"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name implements [Provider].
func (l *lrcLibProvider) name() Source {
	return "api-lrclib"
}

func cacheKey(track TrackInfo) string {
	h := sha256.New()
	h.Write([]byte(track.Artist + "|" + track.Title + "|" + track.Album + "|" + track.Name + "|" + fmt.Sprint(track.Duration.Seconds())))
	return hex.EncodeToString(h.Sum(nil))
}

// Find implements [Provider].
func (l *lrcLibProvider) find(track TrackInfo) (Lyrics, error) {
	if !l.enabled {
		return Lyrics{}, errors.New("lrclib provider is disabled")
	}
	key := cacheKey(track)
	cacheFile := filepath.Join(l.cacheDir, key+".json")

	if l.cacheEnabled {
		if data, err := os.ReadFile(cacheFile); err == nil {
			var cached Lyrics
			if err := json.Unmarshal(data, &cached); err == nil {
				return cached, nil
			}
		}
	}

	var (
		res Lyrics
		err error
	)
	if track.Title == "" {
		res, err = l.search(track)
	} else {
		res, err = l.get(track)
	}

	if err == nil && l.cacheEnabled {
		if err := os.MkdirAll(l.cacheDir, 0o755); err == nil {
			if data, err := json.Marshal(res); err == nil {
				_ = os.WriteFile(cacheFile, data, 0o644)
			}
		}
	}

	return res, err
}

func (l *lrcLibProvider) search(track TrackInfo) (Lyrics, error) {
	q := track.Name
	if ext := filepath.Ext(q); ext != "" {
		q = strings.TrimSuffix(q, ext)
	}
	params := url.Values{}
	params.Set("q", q)
	lrcLibURL := fmt.Sprintf("https://lrclib.net/api/search?%s", params.Encode())

	req, err := http.NewRequest(http.MethodGet, lrcLibURL, http.NoBody)
	if err != nil {
		return Lyrics{}, fmt.Errorf("unable to create lrclib request: %w", err)
	}
	req.Header.Add("User-Agent", "tmus")
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return Lyrics{}, fmt.Errorf("request lrclib api failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Lyrics{}, fmt.Errorf("error by lrclib search api, status: %s, url: %s", resp.Status, lrcLibURL)
	}

	var results []lrcLibResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Lyrics{}, fmt.Errorf("decode lrclib search response failed: %w", err)
	}
	if len(results) == 0 {
		return Lyrics{}, fmt.Errorf("no lyrics found for query %q", q)
	}

	return l.parseResponse(results[0], lrcLibURL)
}

func (l *lrcLibProvider) get(track TrackInfo) (Lyrics, error) {
	params := url.Values{}
	params.Set("artist_name", track.Artist)
	params.Set("track_name", track.Title)
	params.Set("album_name", track.Album)
	if track.Duration > 0 {
		params.Set("duration", fmt.Sprintf("%d", int(track.Duration.Seconds())))
	}

	lrcLibURL := fmt.Sprintf("https://lrclib.net/api/get?%s", params.Encode())
	req, err := http.NewRequest(http.MethodGet, lrcLibURL, http.NoBody)
	if err != nil {
		return Lyrics{}, fmt.Errorf("unable to create lrclib request: %w", err)
	}
	req.Header.Add("User-Agent", "tmus")
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return Lyrics{}, fmt.Errorf("request lrclib api failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Lyrics{}, fmt.Errorf("error by lrclib api, status: %s, url: %s", resp.Status, lrcLibURL)
	}

	var result lrcLibResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Lyrics{}, fmt.Errorf("decode lrclib response failed: %w", err)
	}

	return l.parseResponse(result, lrcLibURL)
}

func (l *lrcLibProvider) parseResponse(result lrcLibResponse, lrcLibURL string) (Lyrics, error) {
	if lines, timed := parse(result.SyncedLyrics); len(lines) > 0 {
		lex := Lyrics{
			Lines:      lines,
			Timed:      timed,
			Raw:        result.SyncedLyrics,
			Source:     l.name(),
			SourcePath: lrcLibURL,
		}
		return lex, nil
	}

	if lines, timed := parse(result.PlainLyrics); len(lines) > 0 {
		lex := Lyrics{
			Lines:      lines,
			Timed:      timed,
			Raw:        result.PlainLyrics,
			Source:     l.name(),
			SourcePath: lrcLibURL,
		}
		return lex, nil
	}

	return Lyrics{}, fmt.Errorf("cannot interpret lrclib record with id %d", result.ID)
}

type lrcLibResponse struct {
	ID           uint64 `json:"id"`
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}
