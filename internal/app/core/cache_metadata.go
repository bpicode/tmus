package core

import (
	"container/list"
	"sync"
)

const (
	metadataCacheMaxEntries = 128
	metadataCacheMaxBytes   = 32 * 1024 * 1024
)

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

func newMetadataCache() *metadataCache {
	return &metadataCache{
		maxEntries: metadataCacheMaxEntries,
		maxBytes:   metadataCacheMaxBytes,
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
