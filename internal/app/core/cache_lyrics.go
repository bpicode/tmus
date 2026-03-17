package core

import (
	"container/list"
	"sync"

	"github.com/bpicode/tmus/internal/app/lyrics"
)

const (
	lyricsCacheMaxEntries = 256
	lyricsCacheMaxBytes   = 8 * 1024 * 1024
)

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

func newLyricsCache() *lyricsCache {
	return &lyricsCache{
		maxEntries: lyricsCacheMaxEntries,
		maxBytes:   lyricsCacheMaxBytes,
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
