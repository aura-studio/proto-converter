package converter

import (
	"os"
	"path/filepath"
	"strings"
)

// DepResolver 负责从初始种子递归解析 import，得到可达 proto 集。
type DepResolver struct{}

func (DepResolver) CollectWithImports(seeds []protoItem) ([]protoItem, error) {
	roots := []string{".", filepath.FromSlash("external/proto/cli"), filepath.FromSlash("external/proto/shared")}
	seen := map[string]protoItem{}
	var queue []protoItem
	push := func(it protoItem) {
		key := strings.ToLower(it.Base)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = it
		queue = append(queue, it)
	}
	for _, it := range seeds {
		push(it)
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		path := cur.Path
		if !exists(path) {
			candidates := []string{}
			if cur.Dir != "" {
				candidates = append(candidates, filepath.Join(cur.Dir, cur.Base))
			}
			for _, r := range roots {
				candidates = append(candidates, filepath.Join(r, cur.Base))
			}
			for _, c := range candidates {
				if exists(c) {
					path = c
					break
				}
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		matches := importRe.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			imp := strings.TrimSpace(m[1])
			var found string
			if cur.Dir != "" {
				p := filepath.Join(cur.Dir, imp)
				if exists(p) {
					found = p
				}
			}
			if found == "" {
				for _, r := range roots {
					p := filepath.Join(r, imp)
					if exists(p) {
						found = p
						break
					}
				}
			}
			if found == "" {
				continue
			}
			if it, err := normalizeItem(found); err == nil {
				push(it)
			}
		}
	}

	out := make([]protoItem, 0, len(seen))
	for _, it := range seen {
		out = append(out, it)
	}
	for i := 0; i < len(out); i++ { // simple sort by Base
		for j := i + 1; j < len(out); j++ {
			if strings.ToLower(out[j].Base) < strings.ToLower(out[i].Base) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}
