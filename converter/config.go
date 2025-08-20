package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// FileRule describes one seed proto file and its kept fields.
type FileRule struct {
	File string   `yaml:"file"`
	Keep []string `yaml:"keep"`
}

// TypeRule describes fields to keep for a specific message type.
type TypeRule struct {
	Type string   `yaml:"type"`
	Keep []string `yaml:"keep"`
}

// ProtoExportConfig is a flattened config mapped from v2 import/export YAML.
type ProtoExportConfig struct {
	Files         []FileRule `yaml:"files"`
	Types         []TypeRule `yaml:"types"`
	Protogen      string     `yaml:"protogen"`
	ExportDir     string     `yaml:"exportDir"`
	ImportDir     string     `yaml:"importDir"`
	Language      string     `yaml:"language"`
	Namespace     string     `yaml:"namespace"`
	FileNameCase  string     `yaml:"fileNameCase"`
	FieldNameCase string     `yaml:"fieldNameCase"`
	Prune         *bool      `yaml:"prune"`
	DryRun        *bool      `yaml:"dryRun"`
}

type importKeep struct {
	Files []FileRule `yaml:"files"`
	Types []TypeRule `yaml:"types"`
}

type importSection struct {
	Dir   string     `yaml:"dir"`
	Prune *bool      `yaml:"prune"`
	Keep  importKeep `yaml:"keep"`
}

type exportSection struct {
	Dir           string `yaml:"dir"`
	Language      string `yaml:"language"`
	Namespace     string `yaml:"namespace"`
	FileNameCase  string `yaml:"fileNameCase"`
	FieldNameCase string `yaml:"fieldNameCase"`
}

type v2Config struct {
	DryRun *bool         `yaml:"dryRun"`
	Import importSection `yaml:"import"`
	Export exportSection `yaml:"export"`
}

func readProtoConfig(path string) (cfg ProtoExportConfig, seeds []protoItem, seedKeep map[string]map[string]struct{}, typeFieldKeep map[string]map[string]struct{}, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("无法打开配置 %s: %w", path, err)
	}

	var v2 v2Config
	if err := yaml.Unmarshal(data, &v2); err == nil {
		if v2.Export.Language != "" || v2.Export.Dir != "" || v2.Import.Dir != "" || len(v2.Import.Keep.Files) > 0 || len(v2.Import.Keep.Types) > 0 || v2.Import.Prune != nil || v2.DryRun != nil {
			cfg = ProtoExportConfig{
				Files:         v2.Import.Keep.Files,
				Types:         v2.Import.Keep.Types,
				ExportDir:     v2.Export.Dir,
				ImportDir:     v2.Import.Dir,
				Language:      v2.Export.Language,
				Namespace:     v2.Export.Namespace,
				FileNameCase:  v2.Export.FileNameCase,
				FieldNameCase: v2.Export.FieldNameCase,
				Prune:         v2.Import.Prune,
				DryRun:        v2.DryRun,
			}
		}
	}
	if len(cfg.Files) == 0 && len(cfg.Types) == 0 && cfg.Language == "" && cfg.ExportDir == "" && cfg.ImportDir == "" {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("仅支持 import/export 结构配置：请参考模板 export_*_proto.yaml")
	}
	if len(cfg.Files) == 0 && len(cfg.Types) == 0 {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("配置为空: 未指定 files 或 types")
	}

	var fileList []string
	rawKeep := map[string][]string{}
	for _, fr := range cfg.Files {
		t := strings.TrimSpace(fr.File)
		if t == "" {
			continue
		}
		fileList = append(fileList, t)
		rawKeep[t] = fr.Keep
	}
	if len(fileList) == 0 {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("配置 files 为空: 需要至少一个种子文件")
	}
	sd, err := (SeedLoader{}).SeedsFromList(fileList)
	if err != nil {
		return ProtoExportConfig{}, nil, nil, nil, err
	}
	seedKeep = map[string]map[string]struct{}{}
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

	typeFieldKeep = map[string]map[string]struct{}{}
	for _, tr := range cfg.Types {
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
	return cfg, sd, seedKeep, typeFieldKeep, nil
}
