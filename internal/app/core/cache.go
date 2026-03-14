package core

import (
	"container/list"
	"os"
	"sync"
	"time"

	"github.com/bpicode/tmus/internal/app/archive"
	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/app/lyrics"
)

const (
	metadataCacheMaxEntries = 128
	metadataCacheMaxBytes   = 32 * 1024 * 1024
	lyricsCacheMaxEntries   = 256
	lyricsCacheMaxBytes     = 8 * 1024 * 1024
)

type fileStat struct {
	modTime time.Time
	size    int64
	ok      bool
}

func statPath(path string) fileStat {
	if path == "" || archive.IsArchivePath(path) {
		return fileStat{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fileStat{}
	}
	return fileStat{
		modTime: info.ModTime(),
		size:    info.Size(),
		ok:      true,
	}
}

func (s fileStat) equal(other fileStat) bool {
	if !s.ok || !other.ok {
		return false
	}
	return s.size == other.size && s.modTime.Equal(other.modTime)
}

type metadataCache struct {
	mu         sync.Mutex
	maxEntries int
	maxBytes   int
	bytes      int
	items      map[string]*list.Element
	order      *list.List
}

type metadataCacheEntry struct {
	key   string
	meta  Metadata
	scope MetadataScope
	size  int
	stat  fileStat
}

func newMetadataCache(maxEntries, maxBytes int) *metadataCache {
	return &metadataCache{
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
		items:      make(map[string]*list.Element),
		order:      list.New(),
	}
}

func (c *metadataCache) get(path string, scope MetadataScope) (Metadata, bool) {
	if c == nil || path == "" {
		return Metadata{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.items[path]
	if !ok {
		return Metadata{}, false
	}
	entry := elem.Value.(*metadataCacheEntry)
	if scope == MetadataExtended && entry.scope != MetadataExtended {
		return Metadata{}, false
	}
	if entry.stat.ok {
		current := statPath(path)
		if !current.ok || !entry.stat.equal(current) {
			c.remove(elem)
			return Metadata{}, false
		}
	}
	c.order.MoveToFront(elem)
	return entry.meta, true
}

func (c *metadataCache) put(path string, scope MetadataScope, meta Metadata) {
	if c == nil || path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	size := estimateMetadataSize(meta)
	if c.maxBytes > 0 && size > c.maxBytes {
		return
	}
	stat := statPath(path)
	if elem, ok := c.items[path]; ok {
		entry := elem.Value.(*metadataCacheEntry)
		if entry.scope == MetadataExtended && scope == MetadataBasic && entry.stat.equal(stat) {
			c.order.MoveToFront(elem)
			return
		}
		c.bytes -= entry.size
		entry.meta = meta
		entry.scope = scope
		entry.size = size
		entry.stat = stat
		c.bytes += entry.size
		c.order.MoveToFront(elem)
		c.evict()
		return
	}
	entry := &metadataCacheEntry{
		key:   path,
		meta:  meta,
		scope: scope,
		size:  size,
		stat:  stat,
	}
	elem := c.order.PushFront(entry)
	c.items[path] = elem
	c.bytes += entry.size
	c.evict()
}

func (c *metadataCache) remove(elem *list.Element) {
	if c == nil || elem == nil {
		return
	}
	entry := elem.Value.(*metadataCacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
	c.bytes -= entry.size
}

func (c *metadataCache) evict() {
	for {
		if c.maxEntries > 0 && c.order.Len() > c.maxEntries {
			c.remove(c.order.Back())
			continue
		}
		if c.maxBytes > 0 && c.bytes > c.maxBytes {
			c.remove(c.order.Back())
			continue
		}
		break
	}
}

func estimateMetadataSize(meta Metadata) int {
	size := 0
	size += len(meta.Artist)
	size += len(meta.Title)
	size += len(meta.Album)
	size += len(meta.AlbumArtist)
	size += len(meta.Composer)
	size += len(meta.Genre)
	size += len(meta.Comment)
	size += len(meta.Lyrics)
	if meta.Picture != nil {
		size += len(meta.Picture.MIMEType)
		size += len(meta.Picture.Type)
		size += len(meta.Picture.Description)
		size += len(meta.Picture.Data)
	}
	return size
}

func (a *App) readMetadataExtendedCached(path string) (Metadata, error) {
	if meta, ok := a.getCachedMetadata(path, MetadataExtended); ok {
		return meta, nil
	}
	meta, err := library.ReadMetadataExtended(path)
	if err != nil {
		return Metadata{}, err
	}
	a.putCachedMetadata(path, MetadataExtended, meta)
	return meta, nil
}

func (a *App) readLyricsFromTagsCached(path string) (string, error) {
	meta, err := a.readMetadataExtendedCached(path)
	if err != nil {
		return "", err
	}
	return meta.Lyrics, nil
}

func (a *App) getCachedMetadata(path string, scope MetadataScope) (Metadata, bool) {
	if a == nil || a.metadataCache == nil {
		return Metadata{}, false
	}
	return a.metadataCache.get(path, scope)
}

func (a *App) putCachedMetadata(path string, scope MetadataScope, meta Metadata) {
	if a == nil || a.metadataCache == nil {
		return
	}
	a.metadataCache.put(path, scope, meta)
}

func (a *App) getCachedLyrics(path string) (lyrics.Lyrics, bool) {
	if a == nil || a.lyricsCache == nil {
		return lyrics.Lyrics{}, false
	}
	return a.lyricsCache.get(path)
}

func (a *App) putCachedLyrics(path string, data lyrics.Lyrics) {
	if a == nil || a.lyricsCache == nil {
		return
	}
	a.lyricsCache.put(path, data)
}

type lyricsCache struct {
	mu         sync.Mutex
	maxEntries int
	maxBytes   int
	bytes      int
	items      map[string]*list.Element
	order      *list.List
}

type lyricsCacheEntry struct {
	key        string
	sourcePath string
	lyrics     lyrics.Lyrics
	size       int
	stat       fileStat
}

func newLyricsCache(maxEntries, maxBytes int) *lyricsCache {
	return &lyricsCache{
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
		items:      make(map[string]*list.Element),
		order:      list.New(),
	}
}

func (c *lyricsCache) get(path string) (lyrics.Lyrics, bool) {
	if c == nil || path == "" {
		return lyrics.Lyrics{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.items[path]
	if !ok {
		return lyrics.Lyrics{}, false
	}
	entry := elem.Value.(*lyricsCacheEntry)
	if entry.stat.ok {
		checkPath := entry.sourcePath
		if checkPath == "" {
			checkPath = entry.key
		}
		current := statPath(checkPath)
		if !current.ok || !entry.stat.equal(current) {
			c.remove(elem)
			return lyrics.Lyrics{}, false
		}
	}
	c.order.MoveToFront(elem)
	return entry.lyrics, true
}

func (c *lyricsCache) put(path string, data lyrics.Lyrics) {
	if c == nil || path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	size := estimateLyricsSize(data)
	if c.maxBytes > 0 && size > c.maxBytes {
		return
	}
	sourcePath := data.SourcePath
	if sourcePath == "" {
		sourcePath = path
	}
	stat := statPath(sourcePath)
	if elem, ok := c.items[path]; ok {
		entry := elem.Value.(*lyricsCacheEntry)
		c.bytes -= entry.size
		entry.sourcePath = sourcePath
		entry.lyrics = data
		entry.size = size
		entry.stat = stat
		c.bytes += entry.size
		c.order.MoveToFront(elem)
		c.evict()
		return
	}
	entry := &lyricsCacheEntry{
		key:        path,
		sourcePath: sourcePath,
		lyrics:     data,
		size:       size,
		stat:       stat,
	}
	elem := c.order.PushFront(entry)
	c.items[path] = elem
	c.bytes += entry.size
	c.evict()
}

func (c *lyricsCache) remove(elem *list.Element) {
	if c == nil || elem == nil {
		return
	}
	entry := elem.Value.(*lyricsCacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
	c.bytes -= entry.size
}

func (c *lyricsCache) evict() {
	for {
		if c.maxEntries > 0 && c.order.Len() > c.maxEntries {
			c.remove(c.order.Back())
			continue
		}
		if c.maxBytes > 0 && c.bytes > c.maxBytes {
			c.remove(c.order.Back())
			continue
		}
		break
	}
}

func estimateLyricsSize(data lyrics.Lyrics) int {
	size := len(data.Raw)
	size += len(data.SourcePath)
	for _, line := range data.Lines {
		size += len(line.Text)
		size += 16
	}
	return size
}
