package converter

import (
	"path/filepath"
	"strings"
)

// SeedLoader 负责从配置列表派生初始种子 proto 条目
type SeedLoader struct{}

// SeedsFromList 将配置行标准化为 protoItem 列表；裸文件名默认 external/proto/cli 下
func (SeedLoader) SeedsFromList(list []string) ([]protoItem, error) {
	var seeds []protoItem
	for _, s := range list {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		if !strings.ContainsAny(t, "/\\") { // 裸名
			t = filepath.FromSlash(filepath.Join("external/proto/cli", t))
			if !strings.HasSuffix(strings.ToLower(t), ".proto") {
				t += ".proto"
			}
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
