package converter

import "github.com/aura-studio/proto-converter/converter/model"

// Type aliases re-exported from model package to maintain backward compatibility.
type ProtoItem = model.ProtoItem
type PFile = model.PFile
type TopDef = model.TopDef
type DefRef = model.DefRef
type Block = model.Block
type PruneOptions = model.PruneOptions

// Function re-exports from model package.
var (
	NormalizeItem = model.NormalizeItem
	Exists        = model.Exists
	ShortPath     = model.ShortPath
	TrimExt       = model.TrimExt
	EnsureDir     = model.EnsureDir
	BlockBody     = model.BlockBody
)
