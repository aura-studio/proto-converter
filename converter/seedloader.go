package converter

import (
	"strings"
)

// SeedLoader normalizes seed file names and deduplicates by basename.
type SeedLoader struct{}

// SeedsFromList normalizes a list of seed names to protoItems and appends .proto if missing.
func (SeedLoader) SeedsFromList(list []string) ([]protoItem, error) {
	var seeds []protoItem
	for _, s := range list {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(t), ".proto") {
			t += ".proto"
		}
		it, err := normalizeItem(t)
		if err != nil {
			return nil, err
		}
		seeds = append(seeds, it)
	}
	return dedupItems(seeds), nil
}

func dedupItems(items []protoItem) []protoItem {
	seen := map[string]bool{}
	res := make([]protoItem, 0, len(items))
	for _, it := range items {
		key := strings.ToLower(it.Base)
		if seen[key] {
			continue
		}
		seen[key] = true
		res = append(res, it)
	}
	return res
}
