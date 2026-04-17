package parser

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/aura-studio/proto-converter/converter/core/model"
)

// ProtoParser implements the converter.Parser interface for proto file parsing.
type ProtoParser struct{}

// ParseFile reads and parses a .proto file into a PFile structure.
func (p ProtoParser) ParseFile(path string) (*model.PFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	noCom := p.StripComments(content)
	syn := "proto3"
	if m := regexp.MustCompile(`(?m)^\s*syntax\s*=\s*"([^"]+)"\s*;`).FindStringSubmatch(noCom); len(m) == 2 {
		syn = m[1]
	}
	pkg := ""
	if m := regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][\w\.]*?)\s*;`).FindStringSubmatch(noCom); len(m) == 2 {
		pkg = m[1]
	}
	blocks := p.ScanTopLevelBlocks(content)
	defs := make([]model.TopDef, 0, len(blocks))
	for _, bl := range blocks {
		body := model.BlockBody(content, bl)
		refs := p.ExtractTypeRefs(p.StripComments(body))
		defs = append(defs, model.TopDef{Kind: bl.Kind, Name: bl.Name, Text: bl.FullText(content), Refs: refs})
	}
	return &model.PFile{Path: filepath.ToSlash(path), Package: pkg, Syntax: syn, Defs: defs}, nil
}
