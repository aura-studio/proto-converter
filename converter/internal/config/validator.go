package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SeedItem represents a normalized seed file entry within the config package.
type SeedItem struct {
	Path string
	Dir  string
	Base string
}

// Validator is responsible for validating configuration and building seed data.
type Validator struct{}

// Validate checks the completeness and validity of a Config.
func (Validator) Validate(cfg Config) error {
	if cfg.Export.Language == "" && cfg.Export.Dir == "" && cfg.Import.Dir == "" &&
		len(cfg.Import.Keep.Files) == 0 && len(cfg.Import.Keep.Types) == 0 &&
		cfg.Import.Prune == nil && cfg.DryRun == nil {
		return fmt.Errorf("仅支持 import/export 结构配置：请参考模板 export_*_proto.yaml")
	}
	return nil
}

// BuildSeedKeep builds seed items and keep rules from the configuration.
func (Validator) BuildSeedKeep(cfg Config) (
	[]SeedItem,
	map[string]map[string]struct{},
	map[string]map[string]struct{},
	error,
) {
	var fileList []string
	rawKeep := map[string][]string{}
	for _, fr := range cfg.Import.Keep.Files {
		t := strings.TrimSpace(fr.File)
		if t == "" {
			continue
		}
		fileList = append(fileList, t)
		rawKeep[t] = fr.Keep
	}
	if len(fileList) == 0 {
		return nil, nil, nil, fmt.Errorf("配置 files 为空: 需要至少一个种子文件")
	}

	sd := normalizeSeedList(fileList)

	seedKeep := map[string]map[string]struct{}{}
	for _, it := range sd {
		key := filepath.ToSlash(it.Path)
		cands := []string{key, filepath.ToSlash(it.Base), filepath.ToSlash(filepath.Join(it.Dir, it.Base))}
		keepSet := map[string]struct{}{}
		for _, k := range cands {
			if arr, ok := rawKeep[k]; ok {
				for _, n := range arr {
					n = strings.TrimSpace(n)
					if n != "" {
						keepSet[n] = struct{}{}
					}
				}
			}
		}
		if len(keepSet) > 0 {
			seedKeep[key] = keepSet
		}
	}

	typeFieldKeep := map[string]map[string]struct{}{}
	for _, tr := range cfg.Import.Keep.Types {
		tname := strings.TrimSpace(tr.Type)
		if tname == "" {
			continue
		}
		set := map[string]struct{}{}
		for _, f := range tr.Keep {
			f = strings.TrimSpace(f)
			if f != "" {
				set[f] = struct{}{}
			}
		}
		if len(set) > 0 {
			typeFieldKeep[tname] = set
		}
	}

	return sd, seedKeep, typeFieldKeep, nil
}

// normalizeSeedList normalizes seed file names and deduplicates.
func normalizeSeedList(list []string) []SeedItem {
	var seeds []SeedItem
	for _, s := range list {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(t), ".proto") {
			t += ".proto"
		}
		t = strings.TrimPrefix(t, "./")
		t = strings.TrimPrefix(t, "/")
		t = strings.TrimPrefix(t, "\\")
		if t == "" {
			continue
		}
		base := filepath.Base(t)
		dir := filepath.Dir(t)
		if dir == "." {
			dir = ""
		}
		seeds = append(seeds, SeedItem{Path: t, Dir: dir, Base: base})
	}
	// dedup by lowercase base
	seen := map[string]bool{}
	res := make([]SeedItem, 0, len(seeds))
	for _, it := range seeds {
		key := strings.ToLower(it.Base)
		if seen[key] {
			continue
		}
		seen[key] = true
		res = append(res, it)
	}
	return res
}
