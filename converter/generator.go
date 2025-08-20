package converter

import (
	"fmt"
	"path/filepath"
)

// Generator 负责调用外部 protogen 生成 C# 代码。
type Generator struct {
	ProtogenPath string
	OutDir       string
	DryRun       bool
}

// GenFromTemp 基于临时裁剪目录生成（仅使用 tempRoot 作为 proto_path）。
func (g Generator) GenFromTemp(tempRoot string, targets []protoItem) error {
	for _, t := range targets {
		fmt.Printf("[gen] %s (temp:%s)\n", t.Base, shortPath(filepath.Join(tempRoot, t.Path)))
		if g.DryRun {
			continue
		}
		args := buildArgsForTemp(g.OutDir, tempRoot, t)
		if err := run(g.ProtogenPath, args...); err != nil {
			return err
		}
	}
	return nil
}

// GenDirect 对单个 proto 直接生成（多 proto_path 环境）。
func (g Generator) GenDirect(target protoItem, all []protoItem) error {
	fmt.Printf("[gen] %s (%s)\n", target.Base, target.Dir)
	if g.DryRun {
		return nil
	}
	args := buildArgs(g.OutDir, target, all)
	return run(g.ProtogenPath, args...)
}
