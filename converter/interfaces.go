package converter

import (
	"strings"

	"github.com/aura-studio/proto-converter/converter/model"
)

// Parser defines proto file parsing capabilities.
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

// DepResolverIface defines import dependency resolution capabilities.
// Named DepResolverIface to avoid conflict with the existing DepResolver struct.
type DepResolverIface interface {
	CollectWithImportsAndRoots(seeds []model.ProtoItem, importDir string) ([]model.ProtoItem, []model.ProtoItem, error)
}

// Formatter defines output formatting capabilities.
type Formatter interface {
	Sanitize(src string) string
	StripSelfPackageQualifiers(content, selfPkg string) string
	TransformFieldNames(def, caseKind string) string
	WriteLangNamespaceOption(b *strings.Builder, lang, ns string)
}

// DefPruner defines definition-level pruning capabilities.
type DefPruner interface {
	PruneMessageFields(def string, keepSet map[string]struct{}) string
	PruneOneofFields(blk string, keepSet map[string]struct{}) string
	CollectTypeTokens(def string) []string
}

// SeedLoaderIface defines seed file loading capabilities.
// Named SeedLoaderIface to avoid conflict with the existing SeedLoader struct.
type SeedLoaderIface interface {
	SeedsFromList(list []string) ([]model.ProtoItem, error)
}
