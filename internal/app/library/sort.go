package library

import (
	"cmp"
	"slices"
	"strings"
)

func sortEntries(entries []Entry) {
	slices.SortFunc(entries, func(a, b Entry) int {
		if a.IsDir() != b.IsDir() {
			if a.IsDir() {
				return -1
			}
			return 1
		}
		return cmp.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})
}
