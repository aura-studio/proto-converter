package seedloader

import (
	"strings"

	"github.com/aura-studio/proto-converter/converter/model"
)

// SeedLoader normalizes seed file names and deduplicates by basename.
// It implements converter.SeedLoaderIface.
type SeedLoader struct{}

// SeedsFromList normalizes a list of seed names to ProtoItems and appends .proto if missing.
func (SeedLoader) SeedsFromList(list []string) ([]model.ProtoItem, error) {
	var seeds []model.ProtoItem
	for _, s := range list {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(t), ".proto") {
			t += ".proto"
		}
		it, err := model.NormalizeItem(t)
		if err != nil {
			return nil, err
		}
		seeds = append(seeds, it)
	}
	return dedupItems(seeds), nil
}

// dedupItems removes duplicate ProtoItems by lowercase Base name.
func dedupItems(items []model.ProtoItem) []model.ProtoItem {
	seen := map[string]bool{}
	res := make([]model.ProtoItem, 0, len(items))
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
