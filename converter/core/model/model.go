package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProtoItem represents a proto file entry with its path components.
type ProtoItem struct {
	Path string
	Dir  string
	Base string
}

// PFile represents a parsed proto file.
type PFile struct {
	Path    string
	Package string
	Syntax  string
	Defs    []TopDef
}

// TopDef is a top-level definition block (message/enum) with references.
type TopDef struct {
	Kind string
	Name string
	Text string
	Refs []string
}

// DefRef represents a reference to a top-level definition in a specific file.
type DefRef struct {
	File string
	Def  *TopDef
}

// Block represents a scanned top-level block (message or enum) with position info.
type Block struct {
	Kind, Name             string
	Start, BraceStart, End int
}

// FullText returns the full source text of the block from the original source.
func (b Block) FullText(src string) string { return src[b.Start:b.End] }

// BlockBody returns the inner text between the braces of a Block.
func BlockBody(src string, b Block) string {
	if b.BraceStart+1 < b.End {
		return src[b.BraceStart+1 : b.End-1]
	}
	return ""
}

// PruneOptions encapsulates configuration parameters for the pruning process.
type PruneOptions struct {
	InDir         string
	OutDir        string
	Namespace     string
	Language      string
	FileNameCase  string
	FieldNameCase string
	DryRun        bool
}

// NormalizeItem normalizes a proto file path string into a ProtoItem.
func NormalizeItem(s string) (ProtoItem, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "\\")
	if s == "" {
		return ProtoItem{}, fmt.Errorf("空的 proto 条目")
	}
	base := filepath.Base(s)
	dir := filepath.Dir(s)
	if dir == "." {
		dir = ""
	}
	return ProtoItem{Path: s, Dir: dir, Base: base}, nil
}

// Exists reports whether the named file or directory exists.
func Exists(p string) bool { _, err := os.Stat(p); return err == nil }

// ShortPath converts a file path to forward-slash notation.
func ShortPath(p string) string { return filepath.ToSlash(p) }

// TrimExt removes the file extension from a filename.
func TrimExt(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

// EnsureDir creates a directory (and parents) or prints a dry-run message.
func EnsureDir(dir string, dry bool) error {
	if dry {
		fmt.Printf("[dry] mkdir -p %s\n", dir)
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
