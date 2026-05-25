package library

import (
	"sort"
	"strings"
)

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if (entries[i].Type() == EntryDir) != (entries[j].Type() == EntryDir) {
			return entries[i].Type() == EntryDir
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
}
