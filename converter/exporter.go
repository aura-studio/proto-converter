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
	if cfg.Export.Dir != "" {
		e.ExportDir = filepath.FromSlash(cfg.Export.Dir)
	} else if e.ExportDir == "" {
		e.ExportDir = "."
	}
	if cfg.Import.Dir != "" {
		e.ImportDir = filepath.FromSlash(cfg.Import.Dir)
	}
	if cfg.Export.Namespace != "" {
		e.Namespace = cfg.Export.Namespace
	}
	if cfg.Export.Language != "" {
		e.Language = strings.ToLower(cfg.Export.Language)
	}
	if cfg.Export.FileNameCase != "" {
		e.FileNameCase = strings.ToLower(cfg.Export.FileNameCase)
	} else if e.FileNameCase == "" {
		e.FileNameCase = "keep"
	}
	if cfg.Export.FieldNameCase != "" {
		e.FieldNameCase = strings.ToLower(cfg.Export.FieldNameCase)
	} else if e.FieldNameCase == "" {
		e.FieldNameCase = "keep"
	}
	switch e.Language {
	case "csharp", "cs", "c#", "golang", "go", "lua":
	case "":
		return fmt.Errorf("配置缺失: language 必填。可选值: csharp/cs/c#、golang/go、lua")
	default:
		return fmt.Errorf("不支持的 language: %s (支持: csharp/cs/c#、golang/go、lua)", e.Language)
	}
	if cfg.Import.Prune != nil {
		e.Prune = *cfg.Import.Prune
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

	if _, _, err := (Pruner{}).BuildPrunedTempProtos(normalized, useSeeds, seedKeep, typeFieldKeep, e.ImportDir, e.ExportDir, e.Namespace, e.Language, e.FileNameCase, e.FieldNameCase, e.DryRun); err != nil {
		return fmt.Errorf("写出转换后的 proto 失败: %w", err)
	}
	return nil
}
