package converter

import (
	"os"
	"path/filepath"
	"strings"
)

// DepResolver resolves proto import dependencies and seed locations.
type DepResolver struct{}

// CollectWithImportsAndRoots resolves seeds to actual files and returns the transitive
// closure of imported proto files along with the resolved seed items.
func (DepResolver) CollectWithImportsAndRoots(seeds []protoItem, importDir string) ([]protoItem, []protoItem, error) {
	roots := []string{}
	for _, it := range seeds {
		if it.Dir != "" {
			roots = append(roots, it.Dir)
		}
	}
	if importDir != "" {
		_ = filepath.WalkDir(importDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				roots = append(roots, path)
			}
			return nil
		})
	}
	_ = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			roots = append(roots, path)
		}
		return nil
	})
	roots = append(roots, ".")

	uniq := func(in []string) []string {
		m := map[string]struct{}{}
		out := make([]string, 0, len(in))
		for _, r := range in {
			r = filepath.Clean(r)
			if _, ok := m[r]; ok {
				continue
			}
			m[r] = struct{}{}
			out = append(out, r)
		}
		return out
	}
	roots = uniq(roots)

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
	var resolvedSeeds []protoItem
	for _, it := range seeds {
		if !exists(it.Path) {
			candidates := []string{}
			if it.Dir != "" {
				candidates = append(candidates, filepath.Join(it.Dir, it.Base))
			}
			for _, r := range roots {
				candidates = append(candidates, filepath.Join(r, it.Base))
			}
			for _, c := range candidates {
				if exists(c) {
					if fixed, err := normalizeItem(c); err == nil {
						it = fixed
					}
					break
				}
			}
		}
		push(it)
		resolvedSeeds = append(resolvedSeeds, it)
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
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if strings.ToLower(out[j].Base) < strings.ToLower(out[i].Base) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, resolvedSeeds, nil
}
