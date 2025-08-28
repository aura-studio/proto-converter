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

type ImportKeep struct {
	Files []FileRule `yaml:"files"`
	Types []TypeRule `yaml:"types"`
}

type ImportSection struct {
	Dir   string     `yaml:"dir"`
	Prune *bool      `yaml:"prune"`
	Keep  ImportKeep `yaml:"keep"`
}

type ExportSection struct {
	Dir           string `yaml:"dir"`
	Language      string `yaml:"language"`
	Namespace     string `yaml:"namespace"`
	FileNameCase  string `yaml:"fileNameCase"`
	FieldNameCase string `yaml:"fieldNameCase"`
}

type Config struct {
	DryRun *bool         `yaml:"dryRun"`
	Import ImportSection `yaml:"import"`
	Export ExportSection `yaml:"export"`
}

func readProtoConfig(path string) (cfg Config, seeds []protoItem, seedKeep map[string]map[string]struct{}, typeFieldKeep map[string]map[string]struct{}, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, nil, nil, nil, fmt.Errorf("无法打开配置 %s: %w", path, err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, nil, nil, nil, fmt.Errorf("配置解析失败: %w", err)
	}

	// 校验配置
	if c.Export.Language == "" && c.Export.Dir == "" && c.Import.Dir == "" && len(c.Import.Keep.Files) == 0 && len(c.Import.Keep.Types) == 0 && c.Import.Prune == nil && c.DryRun == nil {
		return Config{}, nil, nil, nil, fmt.Errorf("仅支持 import/export 结构配置：请参考模板 export_*_proto.yaml")
	}

	var fileList []string
	rawKeep := map[string][]string{}
	for _, fr := range c.Import.Keep.Files {
		t := strings.TrimSpace(fr.File)
		if t == "" {
			continue
		}
		fileList = append(fileList, t)
		rawKeep[t] = fr.Keep
	}
	if len(fileList) == 0 {
		return Config{}, nil, nil, nil, fmt.Errorf("配置 files 为空: 需要至少一个种子文件")
	}
	sd, err := (SeedLoader{}).SeedsFromList(fileList)
	if err != nil {
		return Config{}, nil, nil, nil, err
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
	for _, tr := range c.Import.Keep.Types {
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
	return c, sd, seedKeep, typeFieldKeep, nil
}
