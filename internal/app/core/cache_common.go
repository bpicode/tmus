package core

import (
	"os"
	"time"

	"github.com/bpicode/tmus/internal/app/library"
)

type fileStat struct {
	modTime time.Time
	size    int64
	ok      bool
}

func statPath(path string) fileStat {
	if path == "" {
		return fileStat{}
	}
	entry, err := library.EntryFromPath(path)
	if err != nil {
		return fileStat{}
	}
	filesystemPath, ok := entry.FilesystemPath()
	if !ok || filesystemPath != path {
		return fileStat{}
	}
	info, err := os.Stat(filesystemPath)
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
