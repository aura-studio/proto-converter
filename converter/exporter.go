package converter

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Exporter loads config, resolves dependencies, prunes, and writes proto outputs.
type Exporter struct {
	ConfigPath    string
	ExportDir     string
	ImportDir     string
	Namespace     string
	Language      string
	FileNameCase  string
	FieldNameCase string
	Prune         bool
	DryRun        bool
}

// Run executes export with the current Exporter settings.
func (e *Exporter) Run() error {
	cfg, seeds, seedKeep, typeFieldKeep, err := readProtoConfig(e.ConfigPath)
	if err != nil {
		return err
	}
	if cfg.ExportDir != "" {
		e.ExportDir = filepath.FromSlash(cfg.ExportDir)
	} else if e.ExportDir == "" {
		e.ExportDir = "."
	}
	if cfg.ImportDir != "" {
		e.ImportDir = filepath.FromSlash(cfg.ImportDir)
	}
	if cfg.Namespace != "" {
		e.Namespace = cfg.Namespace
	}
	if cfg.Language != "" {
		e.Language = strings.ToLower(cfg.Language)
	}
	if cfg.FileNameCase != "" {
		e.FileNameCase = strings.ToLower(cfg.FileNameCase)
	} else if e.FileNameCase == "" {
		e.FileNameCase = "camel"
	}
	if cfg.FieldNameCase != "" {
		e.FieldNameCase = strings.ToLower(cfg.FieldNameCase)
	} else if e.FieldNameCase == "" {
		e.FieldNameCase = "camel"
	}
	switch e.Language {
	case "csharp", "cs", "c#", "golang", "go", "lua":
	case "":
		return fmt.Errorf("配置缺失: language 必填。可选值: csharp/cs/c#、golang/go、lua")
	default:
		return fmt.Errorf("不支持的 language: %s (支持: csharp/cs/c#、golang/go、lua)", e.Language)
	}
	if cfg.Prune != nil {
		e.Prune = *cfg.Prune
	} else {
		if !e.Prune {
			e.Prune = true
		}
	}
	if cfg.DryRun != nil {
		e.DryRun = *cfg.DryRun
	}
	normalized, resolvedSeeds, err := (DepResolver{}).CollectWithImportsAndRoots(seeds, e.ImportDir)
	if err != nil {
		return err
	}
	if err := ensureDir(e.ExportDir, e.DryRun); err != nil {
		return err
	}
	if !e.Prune {
		seeds = normalized
		seedKeep = nil
	}
	useSeeds := seeds
	if e.Prune {
		useSeeds = resolvedSeeds
	}

	if _, _, err := (Pruner{}).BuildPrunedTempProtos(normalized, useSeeds, seedKeep, typeFieldKeep, e.ExportDir, e.Namespace, e.Language, e.FileNameCase, e.FieldNameCase, e.DryRun); err != nil {
		return fmt.Errorf("写出转换后的 proto 失败: %w", err)
	}
	return nil
}
