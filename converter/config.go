package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// FileRule 指定文件级裁剪：file 对应的 proto 文件，仅保留 keep 中的顶层定义（message/enum）
type FileRule struct {
	File string   `yaml:"file"`
	Keep []string `yaml:"keep"`
}

// TypeRule 指定类型级裁剪：对某个 message，仅保留部分字段。
// Type 可为短名（Message）或全名（package.Message）。
type TypeRule struct {
	Type string   `yaml:"type"`
	Keep []string `yaml:"keep"`
}

// ProtoExportConfig YAML 根结构
type ProtoExportConfig struct {
	Files []FileRule `yaml:"files"`
	Types []TypeRule `yaml:"types"`
	// CLI 相关参数迁移到配置
	Protogen  string `yaml:"protogen"`
	OutDir    string `yaml:"outDir"`
	ProtoOut  string `yaml:"protoOutDir"`
	Namespace string `yaml:"namespace"`
	Prune     *bool  `yaml:"prune"`
	DryRun    *bool  `yaml:"dryRun"`
}

// readProtoConfig 读取 YAML/TXT 配置
// 返回：
//
//	seeds: 初始种子 proto 列表
//	seedKeep: 针对种子文件需保留的顶层定义名集合（空表示保留全部）
//	typeFieldKeep: 针对特定类型需保留的字段集合
func readProtoConfig(path string) (cfg ProtoExportConfig, seeds []protoItem, seedKeep map[string]map[string]struct{}, typeFieldKeep map[string]map[string]struct{}, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("无法打开配置 %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}
	if len(cfg.Files) == 0 && len(cfg.Types) == 0 {
		return ProtoExportConfig{}, nil, nil, nil, fmt.Errorf("配置为空: 未指定 files 或 types")
	}

	// 文件规则 -> 种子与保留表
	var fileList []string
	rawKeep := map[string][]string{}
	for _, fr := range cfg.Files {
		t := strings.TrimSpace(fr.File)
		if t == "" {
			continue
		}
		// 若包含路径分隔但文件不存在，尝试补全 external/proto 前缀
		if strings.ContainsAny(t, "/\\") && !exists(t) {
			cand := filepath.FromSlash(filepath.Join("external/proto", t))
			if exists(cand) {
				t = cand
			}
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
		// 支持以多种形式匹配
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

	// 类型字段规则
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
