package library

import (
	"sort"
	"strings"
)

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func sortEntries2(entries []Entry2) {
	sort.Slice(entries, func(i, j int) bool {
		if (entries[i].Type() == EntryDir) != (entries[j].Type() == EntryDir) {
			return entries[i].Type() == EntryDir
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
}
