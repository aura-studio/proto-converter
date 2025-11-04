package converter

import (
	"log"
	"os"

	yaml "gopkg.in/yaml.v3"
)

type FileRule struct {
	File string   `yaml:"file"`
	Keep []string `yaml:"keep"`
}

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
	Prune bool       `yaml:"prune"`
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
	DryRun bool          `yaml:"dryRun"`
	Import ImportSection `yaml:"import"`
	Export ExportSection `yaml:"export"`
}

func NewConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Panicf("无法打开配置 %s: %w", path, err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		log.Panicf("配置解析失败: %w", err)
	}

	if c.Export.Language == "" && c.Export.Dir == "" && c.Import.Dir == "" && len(c.Import.Keep.Files) == 0 && len(c.Import.Keep.Types) == 0 && c.Import.Prune == nil && c.DryRun == nil {
		log.Panicf("参数错误，参考结构配置：template.proto.yaml")
	}

	return c
}
