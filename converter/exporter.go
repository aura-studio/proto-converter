package converter

import (
	"fmt"
	"path/filepath"
)

// Exporter 按配置执行导出流程。
type Exporter struct {
	// inputs
	ConfigPath   string
	ProtogenPath string
	OutDir       string
	ProtoOutDir  string
	Namespace    string
	Prune        bool
	DryRun       bool
}

// Run 执行导出流程（基于 -config 配置）。
func (e *Exporter) Run() error {
	cfg, seeds, seedKeep, typeFieldKeep, err := readProtoConfig(e.ConfigPath)
	if err != nil {
		return err
	}
	// 将 YAML 中的 cli 字段填充到 Exporter（提供默认值）
	if cfg.Protogen != "" {
		e.ProtogenPath = cfg.Protogen
	} else if e.ProtogenPath == "" {
		e.ProtogenPath = filepath.FromSlash("bin/protogen/protogen.exe")
	}
	if cfg.OutDir != "" {
		e.OutDir = filepath.FromSlash(cfg.OutDir)
	} else if e.OutDir == "" {
		e.OutDir = filepath.FromSlash("external/CSharpExport/Proto/Gen")
	}
	if cfg.ProtoOut != "" {
		e.ProtoOutDir = filepath.FromSlash(cfg.ProtoOut)
	} else if e.ProtoOutDir == "" {
		e.ProtoOutDir = filepath.FromSlash("external/CSharpExport/Proto/Schema")
	}
	if cfg.Namespace != "" {
		e.Namespace = cfg.Namespace
	} else if e.Namespace == "" {
		e.Namespace = "Export.Proto"
	}
	if cfg.Prune != nil {
		e.Prune = *cfg.Prune
	} else {
		// 默认为开启裁剪
		if !e.Prune {
			e.Prune = true
		}
	}
	if cfg.DryRun != nil {
		e.DryRun = *cfg.DryRun
	}
	if !exists(e.ProtogenPath) {
		return fmt.Errorf("protogen 不存在: %s", e.ProtogenPath)
	}
	normalized, err := (DepResolver{}).CollectWithImports(seeds)
	if err != nil {
		return err
	}

	if err := ensureDir(e.OutDir, e.DryRun); err != nil {
		return err
	}
	if err := ensureDir(e.ProtoOutDir, e.DryRun); err != nil {
		return err
	}

	var genTargets []protoItem
	if e.Prune {
		tempRoot, targets, err := (Pruner{}).BuildPrunedTempProtos(normalized, seeds, seedKeep, typeFieldKeep, e.ProtoOutDir, e.Namespace, e.DryRun)
		if err != nil {
			return err
		}
		gen := Generator{ProtogenPath: e.ProtogenPath, OutDir: e.OutDir, DryRun: e.DryRun}
		if err := gen.GenFromTemp(tempRoot, targets); err != nil {
			return err
		}
		genTargets = targets
	} else {
		gen := Generator{ProtogenPath: e.ProtogenPath, OutDir: e.OutDir, DryRun: e.DryRun}
		for _, it := range normalized {
			if err := gen.GenDirect(it, normalized); err != nil {
				return err
			}
		}
		genTargets = normalized
	}

	expected := make(map[string]struct{}, len(genTargets))
	for _, it := range genTargets {
		name := snakeToCamel(trimExt(it.Base)) + ".cs"
		expected[name] = struct{}{}
	}
	cl := Cleaner{OutDir: e.OutDir, DryRun: e.DryRun}
	if err := cl.RenameToCamelCase(); err != nil {
		return err
	}
	return cl.DeleteExtras(expected)
}
