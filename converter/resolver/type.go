package resolver

import (
	"strings"

	"github.com/aura-studio/proto-converter/converter/model"
)

// TypeResolver resolves type references using indexed parsed proto files.
type TypeResolver struct {
	parsed      map[string]*model.PFile
	fullIndex   map[string]model.DefRef
	simpleIndex map[string][]model.DefRef
	pkgs        map[string]struct{}
}

// NewTypeResolver creates a new TypeResolver instance.
func NewTypeResolver() *TypeResolver {
	return &TypeResolver{}
}

// BuildIndex builds the full and simple type-name-to-definition indexes.
func (r *TypeResolver) BuildIndex(parsed map[string]*model.PFile) (map[string]model.DefRef, map[string][]model.DefRef) {
	r.parsed = parsed
	r.fullIndex = map[string]model.DefRef{}
	r.simpleIndex = map[string][]model.DefRef{}
	r.pkgs = map[string]struct{}{}
	for filePath, pf := range parsed {
		if pf.Package != "" {
			r.pkgs[pf.Package] = struct{}{}
		}
		for i := range pf.Defs {
			d := &pf.Defs[i]
			if pf.Package != "" {
				r.fullIndex[pf.Package+"."+d.Name] = model.DefRef{File: filePath, Def: d}
			}
			r.simpleIndex[d.Name] = append(r.simpleIndex[d.Name], model.DefRef{File: filePath, Def: d})
		}
	}
	return r.fullIndex, r.simpleIndex
}

// Resolve resolves a type token to a concrete definition reference.
func (r *TypeResolver) Resolve(curFile, curPkg, token string) (model.DefRef, bool) {
	t := strings.TrimPrefix(strings.TrimSpace(token), ".")
	if t == "" {
		return model.DefRef{}, false
	}
	if _, ok := ScalarTypes[t]; ok {
		return model.DefRef{}, false
	}
	if _, ok := WellKnownTypes[t]; ok {
		return model.DefRef{}, false
	}
	base := BaseName(t)
	if pf := r.parsed[curFile]; pf != nil {
		for i := range pf.Defs {
			if pf.Defs[i].Name == base {
				return model.DefRef{File: curFile, Def: &pf.Defs[i]}, true
			}
		}
	}
	if fqn, ok := r.resolveTop(curPkg, t); ok {
		return r.fullIndex[fqn], true
	}
	if lst, ok := r.simpleIndex[base]; ok && len(lst) == 1 {
		return lst[0], true
	}
	return model.DefRef{}, false
}

// resolveTop resolves a top-level type reference to a fully-qualified name.
func (r *TypeResolver) resolveTop(curPkg, token string) (string, bool) {
	t := strings.TrimPrefix(strings.TrimSpace(token), ".")
	if t == "" {
		return "", false
	}
	if _, ok := ScalarTypes[t]; ok {
		return "", false
	}
	parts := strings.Split(t, ".")
	if len(parts) == 1 {
		if curPkg == "" {
			return "", false
		}
		fqn := curPkg + "." + parts[0]
		if _, ok := r.fullIndex[fqn]; ok {
			return fqn, true
		}
		return "", false
	}
	if _, ok := r.pkgs[parts[0]]; ok {
		fqn := parts[0] + "." + parts[1]
		if _, ok := r.fullIndex[fqn]; ok {
			return fqn, true
		}
	}
	if _, ok := WellKnownTypes[t]; ok {
		return t, false
	}
	return "", false
}

// BaseName extracts the simple name from a possibly qualified type token.
func BaseName(tok string) string {
	t := strings.TrimPrefix(strings.TrimSpace(tok), ".")
	if t == "" {
		return t
	}
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}
