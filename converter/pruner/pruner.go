package pruner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aura-studio/proto-converter/converter/model"
	"github.com/aura-studio/proto-converter/converter/resolver"
)

// Parser defines proto file parsing capabilities (local interface to avoid importing converter).
type Parser interface {
	ParseFile(path string) (*model.PFile, error)
	ScanTopLevelBlocks(src string) []model.Block
	StripComments(src string) string
	ExtractTypeRefs(src string) []string
}

// TypeResolver defines type reference resolution capabilities.
type TypeResolver interface {
	BuildIndex(parsed map[string]*model.PFile) (fullIndex map[string]model.DefRef, simpleIndex map[string][]model.DefRef)
	Resolve(curFile, curPkg, token string) (model.DefRef, bool)
}

// Formatter defines output formatting capabilities.
type Formatter interface {
	Sanitize(src string) string
	StripSelfPackageQualifiers(content, selfPkg string) string
	TransformFieldNames(def, caseKind string) string
	WriteLangNamespaceOption(b *strings.Builder, lang, ns string)
}

// DefPrunerIface defines definition-level pruning capabilities.
type DefPrunerIface interface {
	PruneMessageFields(def string, keepSet map[string]struct{}) string
	PruneOneofFields(blk string, keepSet map[string]struct{}) string
	CollectTypeTokens(def string) []string
}

// Pruner orchestrates the pruning process by composing Parser, TypeResolver, DefPruner, and Formatter.
type Pruner struct {
	Parser   Parser
	Resolver TypeResolver
	DefPrune DefPrunerIface
	Fmt      Formatter
}

// BuildPrunedTempProtos prunes and writes proto files based on seeds and keep rules.
func (p Pruner) BuildPrunedTempProtos(
	all []model.ProtoItem,
	seeds []model.ProtoItem,
	seedKeep map[string]map[string]struct{},
	typeFieldKeep map[string]map[string]struct{},
	opts model.PruneOptions,
) (string, []model.ProtoItem, error) {
	parsed := map[string]*model.PFile{}
	for _, it := range all {
		path := filepath.ToSlash(it.Path)
		pf, err := p.Parser.ParseFile(it.Path)
		if err != nil {
			return "", nil, fmt.Errorf("解析 proto 失败: %s: %w", it.Path, err)
		}
		parsed[path] = pf
	}

	fullIndex, simpleIndex := p.Resolver.BuildIndex(parsed)

	seedSet := map[string]struct{}{}
	for _, s := range seeds {
		seedSet[filepath.ToSlash(s.Path)] = struct{}{}
	}

	selected := map[string]map[string]struct{}{}
	type defRef struct {
		File string
		Def  *model.TopDef
	}
	var queue []defRef
	addDef := func(file string, d *model.TopDef) {
		set := selected[file]
		if set == nil {
			set = map[string]struct{}{}
			selected[file] = set
		}
		if _, ok := set[d.Name]; ok {
			return
		}
		set[d.Name] = struct{}{}
		queue = append(queue, defRef{File: file, Def: d})
	}

	for filePath, pf := range parsed {
		if _, isSeed := seedSet[filePath]; isSeed {
			keepPath, _ := filepath.Rel(opts.InDir, filePath)
			keepPath = filepath.ToSlash(keepPath)
			if keepSet, ok := seedKeep[keepPath]; ok {
				for i := range pf.Defs {
					if _, ok := keepSet[pf.Defs[i].Name]; ok {
						addDef(filePath, &pf.Defs[i])
					}
				}
			} else {
				for i := range pf.Defs {
					addDef(filePath, &pf.Defs[i])
				}
			}
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curPkg := parsed[cur.File].Package
		srcData, err := os.ReadFile(cur.File)
		if err != nil {
			return "", nil, err
		}
		defTxt := ExtractOriginalBlock(srcData, cur.Def.Text)
		if keepSet := ResolveTypeKeepSet(typeFieldKeep, curPkg, cur.Def.Name); keepSet != nil && strings.TrimSpace(cur.Def.Kind) == "message" {
			defTxt = p.DefPrune.PruneMessageFields(defTxt, keepSet)
		}
		for _, tok := range p.DefPrune.CollectTypeTokens(defTxt) {
			if dr, ok := p.Resolver.Resolve(cur.File, curPkg, tok); ok {
				addDef(dr.File, dr.Def)
			}
		}
	}

	tempRoot := filepath.FromSlash(opts.OutDir)
	if opts.DryRun {
		fmt.Printf("[dry] prepare pruned output in %s\n", tempRoot)
	} else if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", nil, err
	}

	// Use local resolveDef closure for cross-import resolution
	resolveDef := func(curFile, curPkg, token string) (model.DefRef, bool) {
		return p.Resolver.Resolve(curFile, curPkg, token)
	}

	_ = fullIndex
	_ = simpleIndex

	var targets []model.ProtoItem
	for filePath, pf := range parsed {
		base := filepath.Base(filePath)
		rel := model.ToCase(model.TrimExt(base), opts.FileNameCase) + ".proto"
		dstPath := filepath.Join(tempRoot, rel)
		if opts.DryRun {
			fmt.Printf("[dry] mkdir -p %s\n", filepath.Dir(dstPath))
		} else {
			_ = os.MkdirAll(filepath.Dir(dstPath), 0o755)
		}

		chosen := selected[filePath]
		if len(chosen) == 0 {
			if opts.DryRun {
				fmt.Printf("[dry] write stub %s\n", model.ShortPath(dstPath))
			} else {
				var b strings.Builder
				if pf.Syntax != "" {
					b.WriteString("syntax = \"" + pf.Syntax + "\";\n\n")
				} else {
					b.WriteString("syntax = \"proto3\";\n\n")
				}
				if pf.Package != "" {
					b.WriteString("package " + pf.Package + ";\n\n")
				}
				if opts.Namespace != "" {
					p.Fmt.WriteLangNamespaceOption(&b, opts.Language, opts.Namespace)
					b.WriteString("\n\n")
				}
				outTxt := p.Fmt.Sanitize(b.String())
				if err := os.WriteFile(dstPath, []byte(outTxt), 0o644); err != nil {
					return "", nil, err
				}
			}
			continue
		}

		prunedDefs := []string{}
		if opts.DryRun {
			fmt.Printf("[dry] write pruned %s\n", model.ShortPath(dstPath))
		} else {
			srcData, err := os.ReadFile(filePath)
			if err != nil {
				return "", nil, err
			}
			for _, d := range pf.Defs {
				if _, ok := chosen[d.Name]; !ok {
					continue
				}
				def := ExtractOriginalBlock(srcData, d.Text)
				if keepSet := ResolveTypeKeepSet(typeFieldKeep, pf.Package, d.Name); keepSet != nil && strings.TrimSpace(d.Kind) == "message" {
					def = p.DefPrune.PruneMessageFields(def, keepSet)
				}
				def = p.Fmt.StripSelfPackageQualifiers(def, pf.Package)
				if strings.TrimSpace(d.Kind) == "message" {
					def = p.Fmt.TransformFieldNames(def, opts.FieldNameCase)
				}
				prunedDefs = append(prunedDefs, def)
			}
			presentBase := map[string]struct{}{}
			for _, def := range prunedDefs {
				for _, tok := range p.DefPrune.CollectTypeTokens(def) {
					presentBase[baseName(tok)] = struct{}{}
				}
			}
			crossImports := map[string]struct{}{}
			googleImports := map[string]struct{}{}
			for i := range pf.Defs {
				d := &pf.Defs[i]
				if _, ok := chosen[d.Name]; !ok {
					continue
				}
				for _, tok := range d.Refs {
					tokTrim := strings.TrimPrefix(strings.TrimSpace(tok), ".")
					if _, ok := presentBase[baseName(tokTrim)]; !ok {
						continue
					}
					if imp, ok := resolver.WellKnownTypes[tokTrim]; ok {
						googleImports[imp] = struct{}{}
						continue
					}
					if dr, ok := resolveDef(filePath, pf.Package, tokTrim); ok {
						if dr.File != filePath {
							crossImports[dr.File] = struct{}{}
						}
					}
				}
			}

			var b strings.Builder
			if pf.Syntax != "" {
				b.WriteString("syntax = \"" + pf.Syntax + "\";\n\n")
			} else {
				b.WriteString("syntax = \"proto3\";\n\n")
			}
			if pf.Package != "" {
				b.WriteString("package " + pf.Package + ";\n\n")
			}
			for imp := range crossImports {
				impName := model.ToCase(model.TrimExt(filepath.Base(imp)), opts.FileNameCase) + ".proto"
				b.WriteString("import \"" + impName + "\";\n")
			}
			for imp := range googleImports {
				b.WriteString("import \"" + imp + "\";\n")
			}
			if len(crossImports) > 0 || len(googleImports) > 0 {
				b.WriteString("\n")
			}
			if opts.Namespace != "" {
				p.Fmt.WriteLangNamespaceOption(&b, opts.Language, opts.Namespace)
				b.WriteString("\n\n")
			}
			for _, def := range prunedDefs {
				b.WriteString(def)
				b.WriteString("\n\n")
			}
			outTxt := p.Fmt.Sanitize(b.String())
			if err := os.WriteFile(dstPath, []byte(outTxt), 0o644); err != nil {
				return "", nil, err
			}
		}
		targets = append(targets, model.ProtoItem{Path: rel, Dir: "", Base: rel})
	}

	return tempRoot, targets, nil
}

// baseName extracts the simple name from a possibly qualified type token.
func baseName(tok string) string {
	t := strings.TrimPrefix(strings.TrimSpace(tok), ".")
	if t == "" {
		return t
	}
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}
