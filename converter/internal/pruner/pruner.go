package pruner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aura-studio/proto-converter/converter/core/contract"
	"github.com/aura-studio/proto-converter/converter/core/model"
	"github.com/aura-studio/proto-converter/converter/core/util"
	"github.com/aura-studio/proto-converter/converter/internal/resolver"
)

// Pruner orchestrates the pruning process by composing Parser, TypeResolver, DefPruner, and Formatter.
type Pruner struct {
	Parser   contract.Parser
	Resolver contract.TypeResolver
	DefPrune contract.DefPruner
	Fmt      contract.Formatter
}

// parseResult encapsulates the output of the parsing phase.
type parseResult struct {
	parsed      map[string]*model.PFile
	fullIndex   map[string]model.DefRef
	simpleIndex map[string][]model.DefRef
}

// BuildPrunedTempProtos prunes and writes proto files based on seeds and keep rules.
func (p Pruner) BuildPrunedTempProtos(
	all []model.ProtoItem,
	seeds []model.ProtoItem,
	seedKeep map[string]map[string]struct{},
	typeFieldKeep map[string]map[string]struct{},
	opts model.PruneOptions,
) (string, []model.ProtoItem, error) {
	// Phase 1: parse and build indexes
	pr, err := p.parseAndIndex(all)
	if err != nil {
		return "", nil, err
	}

	// Phase 2: BFS dependency tracking
	selected, err := p.collectSelectedDefs(pr, seeds, seedKeep, typeFieldKeep, opts.InDir)
	if err != nil {
		return "", nil, err
	}

	// Phase 3: assemble and write output
	return p.assembleAndWrite(pr, selected, typeFieldKeep, opts)
}

// parseAndIndex parses all proto files and builds type indexes.
func (p Pruner) parseAndIndex(all []model.ProtoItem) (parseResult, error) {
	parsed := map[string]*model.PFile{}
	for _, it := range all {
		path := filepath.ToSlash(it.Path)
		pf, err := p.Parser.ParseFile(it.Path)
		if err != nil {
			return parseResult{}, fmt.Errorf("解析 proto 失败: %s: %w", it.Path, err)
		}
		parsed[path] = pf
	}
	fullIndex, simpleIndex := p.Resolver.BuildIndex(parsed)
	return parseResult{parsed: parsed, fullIndex: fullIndex, simpleIndex: simpleIndex}, nil
}

// collectSelectedDefs performs BFS dependency tracking from seed definitions,
// collecting all definitions that need to be kept.
func (p Pruner) collectSelectedDefs(
	pr parseResult,
	seeds []model.ProtoItem,
	seedKeep map[string]map[string]struct{},
	typeFieldKeep map[string]map[string]struct{},
	inDir string,
) (map[string]map[string]struct{}, error) {
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

	for filePath, pf := range pr.parsed {
		if _, isSeed := seedSet[filePath]; !isSeed {
			continue
		}
		keepPath, _ := filepath.Rel(inDir, filePath)
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

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curPkg := pr.parsed[cur.File].Package
		srcData, err := os.ReadFile(cur.File)
		if err != nil {
			return nil, err
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

	return selected, nil
}

// assembleAndWrite assembles output files from selected definitions and writes them to disk.
func (p Pruner) assembleAndWrite(
	pr parseResult,
	selected map[string]map[string]struct{},
	typeFieldKeep map[string]map[string]struct{},
	opts model.PruneOptions,
) (string, []model.ProtoItem, error) {
	tempRoot := filepath.FromSlash(opts.OutDir)
	if opts.DryRun {
		fmt.Printf("[dry] prepare pruned output in %s\n", tempRoot)
	} else if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", nil, err
	}

	_ = pr.fullIndex
	_ = pr.simpleIndex

	var targets []model.ProtoItem
	for filePath, pf := range pr.parsed {
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
			if err := p.writeStubFile(pf, dstPath, opts); err != nil {
				return "", nil, err
			}
			continue
		}

		if err := p.writePrunedFile(filePath, pf, chosen, typeFieldKeep, dstPath, opts); err != nil {
			return "", nil, err
		}
		targets = append(targets, model.ProtoItem{Path: rel, Dir: "", Base: rel})
	}

	return tempRoot, targets, nil
}

// writeStubFile writes a minimal stub proto file (syntax + package + namespace).
func (p Pruner) writeStubFile(pf *model.PFile, dstPath string, opts model.PruneOptions) error {
	if opts.DryRun {
		fmt.Printf("[dry] write stub %s\n", model.ShortPath(dstPath))
		return nil
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
	if opts.Namespace != "" {
		p.Fmt.WriteLangNamespaceOption(&b, opts.Language, opts.Namespace)
		b.WriteString("\n\n")
	}
	outTxt := p.Fmt.Sanitize(b.String())
	return os.WriteFile(dstPath, []byte(outTxt), 0o644)
}

// writePrunedFile writes a pruned proto file with selected definitions.
func (p Pruner) writePrunedFile(
	filePath string,
	pf *model.PFile,
	chosen map[string]struct{},
	typeFieldKeep map[string]map[string]struct{},
	dstPath string,
	opts model.PruneOptions,
) error {
	if opts.DryRun {
		fmt.Printf("[dry] write pruned %s\n", model.ShortPath(dstPath))
		return nil
	}
	srcData, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	prunedDefs := []string{}
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

	crossImports, googleImports := p.resolveImports(filePath, pf, chosen, prunedDefs)

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
	return os.WriteFile(dstPath, []byte(outTxt), 0o644)
}

// resolveImports determines cross-file and well-known type imports for a pruned file.
func (p Pruner) resolveImports(
	filePath string,
	pf *model.PFile,
	chosen map[string]struct{},
	prunedDefs []string,
) (crossImports, googleImports map[string]struct{}) {
	presentBase := map[string]struct{}{}
	for _, def := range prunedDefs {
		for _, tok := range p.DefPrune.CollectTypeTokens(def) {
			presentBase[util.BaseName(tok)] = struct{}{}
		}
	}
	crossImports = map[string]struct{}{}
	googleImports = map[string]struct{}{}
	for i := range pf.Defs {
		d := &pf.Defs[i]
		if _, ok := chosen[d.Name]; !ok {
			continue
		}
		for _, tok := range d.Refs {
			tokTrim := strings.TrimPrefix(strings.TrimSpace(tok), ".")
			if _, ok := presentBase[util.BaseName(tokTrim)]; !ok {
				continue
			}
			if imp, ok := resolver.WellKnownTypes[tokTrim]; ok {
				googleImports[imp] = struct{}{}
				continue
			}
			if dr, ok := p.Resolver.Resolve(filePath, pf.Package, tokTrim); ok {
				if dr.File != filePath {
					crossImports[dr.File] = struct{}{}
				}
			}
		}
	}
	return crossImports, googleImports
}
