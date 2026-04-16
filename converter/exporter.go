package converter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aura-studio/proto-converter/converter/config"
	"github.com/aura-studio/proto-converter/converter/formatter"
	"github.com/aura-studio/proto-converter/converter/model"
	"github.com/aura-studio/proto-converter/converter/parser"
	"github.com/aura-studio/proto-converter/converter/pruner"
	"github.com/aura-studio/proto-converter/converter/resolver"
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

	// Optional dependency injection
	parser       Parser
	typeResolver TypeResolver
	formatter    Formatter
	defPruner    DefPruner
	depResolver  DepResolverIface
}

// ExporterOption configures an Exporter via functional options.
type ExporterOption func(*Exporter)

// NewExporter creates an Exporter with optional dependency injection.
func NewExporter(opts ...ExporterOption) *Exporter {
	e := &Exporter{}
	for _, o := range opts {
		o(e)
	}
	return e
}

func WithParser(p Parser) ExporterOption             { return func(e *Exporter) { e.parser = p } }
func WithTypeResolver(r TypeResolver) ExporterOption { return func(e *Exporter) { e.typeResolver = r } }
func WithFormatter(f Formatter) ExporterOption       { return func(e *Exporter) { e.formatter = f } }
func WithDefPruner(d DefPruner) ExporterOption       { return func(e *Exporter) { e.defPruner = d } }

// Run executes export with the current Exporter settings.
func (e *Exporter) Run() error {
	p := e.parser
	if p == nil {
		p = parser.ProtoParser{}
	}
	tr := e.typeResolver
	if tr == nil {
		tr = resolver.NewTypeResolver()
	}
	f := e.formatter
	if f == nil {
		f = formatter.OutputFormatter{Parser: p}
	}
	dp := e.defPruner
	if dp == nil {
		dp = pruner.DefinitionPruner{}
	}

	cfg, err := (config.Loader{}).Load(e.ConfigPath)
	if err != nil {
		return err
	}
	if err := (config.Validator{}).Validate(cfg); err != nil {
		return err
	}
	rawSeeds, seedKeep, typeFieldKeep, err := (config.Validator{}).BuildSeedKeep(cfg)
	if err != nil {
		return err
	}
	// Convert config.SeedItem to model.ProtoItem
	seeds := make([]model.ProtoItem, len(rawSeeds))
	for i, s := range rawSeeds {
		seeds[i] = model.ProtoItem{Path: s.Path, Dir: s.Dir, Base: s.Base}
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

	normalized, resolvedSeeds, err := (resolver.DepResolver{}).CollectWithImportsAndRoots(seeds, e.ImportDir)
	if err != nil {
		return err
	}
	if err := model.EnsureDir(e.ExportDir, e.DryRun); err != nil {
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

	prunr := pruner.Pruner{
		Parser:   p,
		Resolver: tr,
		DefPrune: dp,
		Fmt:      f,
	}
	opts := model.PruneOptions{
		InDir:         e.ImportDir,
		OutDir:        e.ExportDir,
		Namespace:     e.Namespace,
		Language:      e.Language,
		FileNameCase:  e.FileNameCase,
		FieldNameCase: e.FieldNameCase,
		DryRun:        e.DryRun,
	}
	if _, _, err := prunr.BuildPrunedTempProtos(normalized, useSeeds, seedKeep, typeFieldKeep, opts); err != nil {
		return fmt.Errorf("写出转换后的 proto 失败: %w", err)
	}
	return nil
}
