package core

import (
	"os"
	"strings"
	"time"
)

type fileStat struct {
	modTime time.Time
	size    int64
	ok      bool
}

func statPath(path string) fileStat {
	if path == "" || strings.Contains(path, "://") {
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
