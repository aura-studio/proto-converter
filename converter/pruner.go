package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pruner selects referenced definitions and writes sanitized proto outputs.
type Pruner struct{}

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

// BuildPrunedTempProtos prunes and writes proto files based on seeds and keep rules.
func (Pruner) BuildPrunedTempProtos(
	all []protoItem,
	seeds []protoItem,
	seedKeep map[string]map[string]struct{},
	typeFieldKeep map[string]map[string]struct{},
	outDir, ns, lang, fileNameCase, fieldNameCase string,
	dry bool,
) (string, []protoItem, error) {
	parsed := map[string]*PFile{}
	pkgs := map[string]struct{}{}
	for _, it := range all {
		p := filepath.ToSlash(it.Path)
		pf, err := parseProtoFile(it.Path)
		if err != nil {
			return "", nil, fmt.Errorf("解析 proto 失败: %s: %w", it.Path, err)
		}
		parsed[p] = pf
		if pf.Package != "" {
			pkgs[pf.Package] = struct{}{}
		}
	}

	type defRef struct {
		File string
		Def  *TopDef
	}
	index := map[string]defRef{}
	simpleIndex := map[string][]defRef{}
	for filePath, pf := range parsed {
		for i := range pf.Defs {
			d := &pf.Defs[i]
			if pf.Package != "" {
				index[pf.Package+"."+d.Name] = defRef{File: filePath, Def: d}
			}
			simpleIndex[d.Name] = append(simpleIndex[d.Name], defRef{File: filePath, Def: d})
		}
	}

	seedSet := map[string]struct{}{}
	for _, s := range seeds {
		seedSet[filepath.ToSlash(s.Path)] = struct{}{}
	}

	selected := map[string]map[string]struct{}{}
	var queue []defRef
	addDef := func(file string, d *TopDef) {
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
			if keepSet, ok := seedKeep[filePath]; ok {
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

	wellKnown := map[string]string{
		"google.protobuf.Timestamp":   "google/protobuf/timestamp.proto",
		"google.protobuf.Duration":    "google/protobuf/duration.proto",
		"google.protobuf.Any":         "google/protobuf/any.proto",
		"google.protobuf.Empty":       "google/protobuf/empty.proto",
		"google.protobuf.Struct":      "google/protobuf/struct.proto",
		"google.protobuf.Value":       "google/protobuf/struct.proto",
		"google.protobuf.ListValue":   "google/protobuf/struct.proto",
		"google.protobuf.Int32Value":  "google/protobuf/wrappers.proto",
		"google.protobuf.Int64Value":  "google/protobuf/wrappers.proto",
		"google.protobuf.StringValue": "google/protobuf/wrappers.proto",
		"google.protobuf.BoolValue":   "google/protobuf/wrappers.proto",
		"google.protobuf.BytesValue":  "google/protobuf/wrappers.proto",
		"google.protobuf.UInt32Value": "google/protobuf/wrappers.proto",
		"google.protobuf.UInt64Value": "google/protobuf/wrappers.proto",
		"google.protobuf.FloatValue":  "google/protobuf/wrappers.proto",
		"google.protobuf.DoubleValue": "google/protobuf/wrappers.proto",
	}
	scalar := map[string]struct{}{"double": {}, "float": {}, "int32": {}, "int64": {}, "uint32": {}, "uint64": {}, "sint32": {}, "sint64": {}, "fixed32": {}, "fixed64": {}, "sfixed32": {}, "sfixed64": {}, "bool": {}, "string": {}, "bytes": {}}

	resolveTop := func(curPkg, token string) (string, bool) {
		t := strings.TrimPrefix(strings.TrimSpace(token), ".")
		if t == "" {
			return "", false
		}
		if _, ok := scalar[t]; ok {
			return "", false
		}
		parts := strings.Split(t, ".")
		if len(parts) == 1 {
			if curPkg == "" {
				return "", false
			}
			fqn := curPkg + "." + parts[0]
			if _, ok := index[fqn]; ok {
				return fqn, true
			}
			return "", false
		}
		if _, ok := pkgs[parts[0]]; ok {
			fqn := parts[0] + "." + parts[1]
			if _, ok := index[fqn]; ok {
				return fqn, true
			}
		}
		if _, ok := wellKnown[t]; ok {
			return t, false
		}
		return "", false
	}

	resolveDef := func(curFile, curPkg, token string) (defRef, bool) {
		t := strings.TrimPrefix(strings.TrimSpace(token), ".")
		if t == "" {
			return defRef{}, false
		}
		if _, ok := scalar[t]; ok {
			return defRef{}, false
		}
		if _, ok := wellKnown[t]; ok {
			return defRef{}, false
		}
		base := baseName(t)
		if pf := parsed[curFile]; pf != nil {
			for i := range pf.Defs {
				if pf.Defs[i].Name == base {
					return defRef{File: curFile, Def: &pf.Defs[i]}, true
				}
			}
		}
		if fqn, ok := resolveTop(curPkg, t); ok {
			return index[fqn], true
		}
		if lst, ok := simpleIndex[base]; ok && len(lst) == 1 {
			return lst[0], true
		}
		return defRef{}, false
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curPkg := parsed[cur.File].Package
		srcData, err := os.ReadFile(cur.File)
		if err != nil {
			return "", nil, err
		}
		defTxt := extractOriginalBlock(srcData, cur.Def.Text)
		if keepSet := resolveTypeKeepSet(typeFieldKeep, curPkg, cur.Def.Name); keepSet != nil && strings.TrimSpace(cur.Def.Kind) == "message" {
			defTxt = pruneMessageFields(defTxt, keepSet)
		}
		for _, tok := range collectTypeTokens(defTxt) {
			if dr, ok := resolveDef(cur.File, curPkg, tok); ok {
				addDef(dr.File, dr.Def)
			}
		}
	}

	tempRoot := filepath.FromSlash(outDir)
	if dry {
		fmt.Printf("[dry] prepare pruned output in %s\n", tempRoot)
	} else if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", nil, err
	}

	var targets []protoItem
	for filePath, pf := range parsed {
		base := filepath.Base(filePath)
		rel := formatProtoFileName(trimExt(base), fileNameCase)
		dstPath := filepath.Join(tempRoot, rel)
		if dry {
			fmt.Printf("[dry] mkdir -p %s\n", filepath.Dir(dstPath))
		} else {
			_ = os.MkdirAll(filepath.Dir(dstPath), 0o755)
		}

		chosen := selected[filePath]
		if len(chosen) == 0 {
			if dry {
				fmt.Printf("[dry] write stub %s\n", shortPath(dstPath))
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
				if ns != "" {
					writeLangNamespaceOption(&b, lang, ns)
					b.WriteString("\n\n")
				}
				outTxt := sanitizeProtoOutput(b.String())
				if err := os.WriteFile(dstPath, []byte(outTxt), 0o644); err != nil {
					return "", nil, err
				}
			}
			continue
		}

		prunedDefs := []string{}
		if dry {
			fmt.Printf("[dry] write pruned %s\n", shortPath(dstPath))
		} else {
			srcData, err := os.ReadFile(filePath)
			if err != nil {
				return "", nil, err
			}
			for _, d := range pf.Defs {
				if _, ok := chosen[d.Name]; !ok {
					continue
				}
				def := extractOriginalBlock(srcData, d.Text)
				if keepSet := resolveTypeKeepSet(typeFieldKeep, pf.Package, d.Name); keepSet != nil && strings.TrimSpace(d.Kind) == "message" {
					def = pruneMessageFields(def, keepSet)
				}
				def = stripSelfPackageQualifiers(def, pf.Package)
				if strings.TrimSpace(d.Kind) == "message" {
					def = transformFieldNames(def, fieldNameCase)
				}
				prunedDefs = append(prunedDefs, def)
			}
			presentBase := map[string]struct{}{}
			for _, def := range prunedDefs {
				for _, tok := range collectTypeTokens(def) {
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
					if imp, ok := wellKnown[tokTrim]; ok {
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
				impName := formatProtoFileName(trimExt(filepath.Base(imp)), fileNameCase)
				b.WriteString("import \"" + impName + "\";\n")
			}
			for imp := range googleImports {
				b.WriteString("import \"" + imp + "\";\n")
			}
			if len(crossImports) > 0 || len(googleImports) > 0 {
				b.WriteString("\n")
			}
			if ns != "" {
				writeLangNamespaceOption(&b, lang, ns)
				b.WriteString("\n\n")
			}
			for _, def := range prunedDefs {
				b.WriteString(def)
				b.WriteString("\n\n")
			}
			outTxt := sanitizeProtoOutput(b.String())
			if err := os.WriteFile(dstPath, []byte(outTxt), 0o644); err != nil {
				return "", nil, err
			}
		}
		targets = append(targets, protoItem{Path: rel, Dir: "", Base: rel})
	}

	return tempRoot, targets, nil
}

func writeLangNamespaceOption(b *strings.Builder, lang, ns string) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "csharp", "cs", "c#":
		b.WriteString("option csharp_namespace = \"" + ns + "\";")
	case "golang", "go":
		b.WriteString("option go_package = \"" + ns + "\";")
	case "lua":

	default:
		b.WriteString("option csharp_namespace = \"" + ns + "\";")
	}
}

func sanitizeProtoOutput(s string) string {
	noCmt := stripCommentsOut(s)
	noRes := dropReservedLines(noCmt)
	compact := normalizeBlankLines(noRes)
	return tightenBlockBlankLines(compact)
}

func stripCommentsOut(s string) string {
	var out strings.Builder
	n := len(s)
	i := 0
	inStr := false
	for i < n {
		c := s[i]
		if inStr {
			out.WriteByte(c)
			if c == '\\' && i+1 < n {
				out.WriteByte(s[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inStr = false
			}
			i++
			continue
		}
		if c == '"' {
			inStr = true
			out.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < n {
			d := s[i+1]
			if d == '/' {
				i += 2
				for i < n && s[i] != '\n' {
					i++
				}
				continue
			}
			if d == '*' {
				i += 2
				for i+1 < n {
					if s[i] == '*' && s[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}
		out.WriteByte(c)
		i++
	}
	return out.String()
}

var reservedLineRe = regexp.MustCompile(`(?i)^\s*reserved\b[^;]*;\s*$`)

func dropReservedLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if reservedLineRe.MatchString(ln) {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

func normalizeBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := func(x string) bool { return strings.TrimSpace(x) == "" }
	prevBlank := true
	for _, ln := range lines {
		if blank(ln) {
			if prevBlank {
				continue
			}
			prevBlank = true
			out = append(out, "")
			continue
		}
		prevBlank = false
		out = append(out, ln)
	}
	for len(out) > 0 && blank(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n") + "\n"
}

func tightenBlockBlankLines(s string) string {
	reAfterOpen := regexp.MustCompile(`\{\r?\n[\t ]*\r?\n`)
	s = reAfterOpen.ReplaceAllString(s, "{\n")
	reBeforeClose := regexp.MustCompile(`\r?\n[\t ]*\r?\n\}`)
	s = reBeforeClose.ReplaceAllString(s, "\n}")
	return s
}

func formatProtoFileName(stem, caseKind string) string {
	switch strings.ToLower(strings.TrimSpace(caseKind)) {
	case "snake":
		return toSnake(stem) + ".proto"
	case "compact":
		return strings.ToLower(removeDelims(stem)) + ".proto"
	case "camel":
		fallthrough
	default:
		return snakeToCamel(stem) + ".proto"
	}
}

func toSnake(s string) string {
	var out []rune
	prevLower := false
	for _, r := range s {
		if r == '-' || r == '.' || r == ' ' {
			r = '_'
		}
		if r >= 'A' && r <= 'Z' {
			if prevLower {
				out = append(out, '_')
			}
			out = append(out, rune(r+'a'-'A'))
			prevLower = false
			continue
		}
		out = append(out, r)
		if r >= 'a' && r <= 'z' {
			prevLower = true
		} else if r == '_' {
			prevLower = false
		}
	}
	res := strings.ReplaceAll(string(out), "__", "_")
	for strings.Contains(res, "__") {
		res = strings.ReplaceAll(res, "__", "_")
	}
	return res
}

func removeDelims(s string) string {
	b := strings.Builder{}
	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func resolveTypeKeepSet(m map[string]map[string]struct{}, pkg, name string) map[string]struct{} {
	if m == nil {
		return nil
	}
	if set, ok := m[name]; ok {
		return set
	}
	if pkg != "" {
		if set, ok := m[pkg+"."+name]; ok {
			return set
		}
	}
	return nil
}

func pruneMessageFields(def string, keepSet map[string]struct{}) string {
	i := strings.Index(def, "{")
	j := strings.LastIndex(def, "}")
	if i < 0 || j <= i {
		return def
	}
	head := def[:i+1]
	body := def[i+1 : j]
	tail := def[j:]

	var out strings.Builder
	out.WriteString(head)

	n := len(body)
	cur := 0
	depth := 0
	for cur < n {
		for cur < n && (body[cur] == ' ' || body[cur] == '\t' || body[cur] == '\r' || body[cur] == '\n') {
			out.WriteByte(body[cur])
			cur++
		}
		if cur >= n {
			break
		}

		if depth == 0 && looksLikeBlock(body[cur:]) {
			kw, start := readKeyword(body, cur)
			if kw == "oneof" {
				_, pos := readIdentAfter(body, start)
				_, blkEnd := findBlock(body, pos)
				if blkEnd <= pos {
					out.WriteString(body[cur:])
					break
				}
				blk := body[cur:blkEnd]
				kept := pruneOneofFields(blk, keepSet)
				if strings.TrimSpace(kept) != "" {
					out.WriteString(kept)
					cur = blkEnd
					continue
				}
				cur = blkEnd
				continue
			}
			if kw == "message" || kw == "enum" || kw == "extend" {
				_, blkEnd := findBlock(body, start)
				if blkEnd <= start {
					out.WriteString(body[cur:])
					break
				}
				out.WriteString(body[cur:blkEnd])
				cur = blkEnd
				continue
			}
		}

		stmtStart := cur
		for cur < n {
			if body[cur] == '"' {
				cur++
				for cur < n {
					if body[cur] == '\\' {
						cur += 2
						continue
					}
					if body[cur] == '"' {
						cur++
						break
					}
					cur++
				}
				continue
			}
			if body[cur] == '{' {
				depth++
				cur++
				continue
			}
			if body[cur] == '}' {
				if depth > 0 {
					depth--
				}
				cur++
				continue
			}
			if body[cur] == ';' && depth == 0 {
				cur++
				break
			}
			cur++
		}
		stmt := body[stmtStart:cur]
		if keepFieldStmt(stmt, keepSet) {
			out.WriteString(stmt)
		}
	}

	out.WriteString(tail)

	res := out.String()
	oi := strings.Index(res, "{")
	oj := strings.LastIndex(res, "}")
	if oi < 0 || oj <= oi {
		return res
	}
	inner := res[oi+1 : oj]
	var nb strings.Builder
	lines := strings.Split(inner, "\n")
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		nb.WriteString(ln)
		nb.WriteByte('\n')
	}
	cleaned := strings.TrimSuffix(nb.String(), "\n")
	if cleaned == "" {
		return res[:oi+1] + "\n" + res[oj:]
	}
	return res[:oi+1] + "\n" + cleaned + "\n" + res[oj:]
}

func looksLikeBlock(s string) bool {
	s = strings.TrimLeft(s, " \t\r\n")
	return strings.HasPrefix(s, "oneof ") || strings.HasPrefix(s, "message ") || strings.HasPrefix(s, "enum ") || strings.HasPrefix(s, "extend ")
}

var (
	reLineComment  = regexp.MustCompile(`(?m)//.*$`)
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reMapType      = regexp.MustCompile(`map\s*<\s*([A-Za-z_][\w\.]*)\s*,\s*([A-Za-z_][\w\.]*)\s*>`)
	reFieldType    = regexp.MustCompile(`(?m)(?:^|[\s{])(?:repeated|optional)?\s*([A-Za-z_][\w\.]*)\s+[A-Za-z_][\w]*\s*=\s*\d+\s*;`)
)

func collectTypeTokens(def string) []string {
	s := reBlockComment.ReplaceAllString(def, "")
	s = reLineComment.ReplaceAllString(s, "")
	toks := map[string]struct{}{}
	for _, m := range reMapType.FindAllStringSubmatch(s, -1) {
		if len(m) >= 3 {
			toks[m[1]] = struct{}{}
			toks[m[2]] = struct{}{}
		}
	}
	for _, m := range reFieldType.FindAllStringSubmatch(s, -1) {
		if len(m) >= 2 {
			toks[m[1]] = struct{}{}
		}
	}
	out := make([]string, 0, len(toks))
	for t := range toks {
		out = append(out, t)
	}
	return out
}

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

func readKeyword(s string, i int) (kw string, next int) {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
		i++
	}
	start := i
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || s[i] == '_') {
		i++
	}
	return s[start:i], i
}

func readIdentAfter(s string, i int) (ident string, next int) {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
		i++
	}
	start := i
	for i < len(s) && (s[i] == '_' || (s[i] >= '0' && s[i] <= '9') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
		i++
	}
	return s[start:i], i
}

func findBlock(s string, i int) (start, end int) {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
		i++
	}
	if i >= len(s) || s[i] != '{' {
		return i, i
	}
	start = i
	i++
	depth := 1
	for i < len(s) {
		if s[i] == '"' {
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if s[i] == '{' {
			depth++
			i++
			continue
		}
		if s[i] == '}' {
			depth--
			i++
			if depth == 0 {
				end = i
				break
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}
		i++
	}
	return start, end
}

func keepFieldStmt(stmt string, keepSet map[string]struct{}) bool {
	s := strings.TrimSpace(stmt)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "oneof ") || strings.HasPrefix(s, "message ") || strings.HasPrefix(s, "enum ") || strings.HasPrefix(s, "extend ") {
		return true
	}
	if !strings.HasSuffix(s, ";") {
		return true
	}
	eq := strings.Index(s, "=")
	if eq <= 0 {
		return true
	}
	left := strings.TrimSpace(s[:eq])
	name := lastIdent(left)
	if name == "" {
		return true
	}
	if len(keepSet) == 0 {
		return true
	}
	_, ok := keepSet[name]
	return ok
}

func lastIdent(s string) string {
	i := len(s) - 1
	for i >= 0 && (s[i] == ' ' || s[i] == '\t') {
		i--
	}
	end := i + 1
	for i >= 0 && (s[i] == '_' || (s[i] >= '0' && s[i] <= '9') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
		i--
	}
	start := i + 1
	if start < 0 || start >= end {
		return ""
	}
	return s[start:end]
}

func pruneOneofFields(blk string, keepSet map[string]struct{}) string {
	lines := strings.Split(blk, "\n")
	if len(keepSet) == 0 {
		return blk
	}
	var out []string
	keptCount := 0
	inBody := false
	for _, ln := range lines {
		s := strings.TrimSpace(ln)
		if strings.HasSuffix(s, "{") && strings.HasPrefix(strings.ToLower(s), "oneof ") {
			inBody = true
			out = append(out, ln)
			continue
		}
		if s == "}" {
			inBody = false
			if keptCount > 0 {
				out = append(out, ln)
			}
			continue
		}
		if !inBody {
			out = append(out, ln)
			continue
		}
		if idx := strings.Index(s, "="); idx > 0 {
			left := strings.TrimSpace(s[:idx])
			name := lastIdent(left)
			if name != "" {
				if _, ok := keepSet[name]; ok {
					out = append(out, ln)
					keptCount++
					continue
				}
				continue
			}
		}
	}
	if keptCount == 0 {
		return ""
	}
	return strings.Join(out, "\n")
}

func parseProtoFile(path string) (*PFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	noCom := stripComments(content)
	syn := "proto3"
	if m := regexp.MustCompile(`(?m)^\s*syntax\s*=\s*"([^"]+)"\s*;`).FindStringSubmatch(noCom); len(m) == 2 {
		syn = m[1]
	}
	pkg := ""
	if m := regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][\w\.]*?)\s*;`).FindStringSubmatch(noCom); len(m) == 2 {
		pkg = m[1]
	}
	blocks := scanTopLevelBlocks(content)
	defs := make([]TopDef, 0, len(blocks))
	for _, bl := range blocks {
		body := blockBody(content, bl)
		refs := extractTypeRefs(stripComments(body))
		defs = append(defs, TopDef{Kind: bl.kind, Name: bl.name, Text: bl.fullText(content), Refs: refs})
	}
	return &PFile{Path: filepath.ToSlash(path), Package: pkg, Syntax: syn, Defs: defs}, nil
}

type block struct {
	kind, name             string
	start, braceStart, end int
}

func (b block) fullText(src string) string { return src[b.start:b.end] }
func blockBody(src string, b block) string {
	if b.braceStart+1 < b.end {
		return src[b.braceStart+1 : b.end-1]
	}
	return ""
}

func scanTopLevelBlocks(src string) []block {
	var out []block
	i, n, depth := 0, len(src), 0
	for i < n {
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			}
			continue
		}
		if src[i] == '"' {
			i++
			for i < n {
				if src[i] == '\\' {
					i += 2
					continue
				}
				if src[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if src[i] == '{' {
			depth++
			i++
			continue
		}
		if src[i] == '}' {
			if depth > 0 {
				depth--
			}
			i++
			continue
		}
		if depth == 0 && isIdentStart(src[i]) {
			start := i
			for i < n && isIdent(src[i]) {
				i++
			}
			kw := src[start:i]
			if kw == "message" || kw == "enum" {
				for i < n && isSpace(src[i]) {
					i++
				}
				nameStart := i
				for i < n && isIdent(src[i]) {
					i++
				}
				name := src[nameStart:i]
				for i < n && src[i] != '{' {
					i++
				}
				if i >= n {
					break
				}
				braceStart := i
				d := 0
				for i < n {
					if src[i] == '"' {
						i++
						for i < n {
							if src[i] == '\\' {
								i += 2
								continue
							}
							if src[i] == '"' {
								i++
								break
							}
							i++
						}
						continue
					}
					if i+1 < n && src[i] == '/' && src[i+1] == '/' {
						i += 2
						for i < n && src[i] != '\n' {
							i++
						}
						continue
					}
					if i+1 < n && src[i] == '/' && src[i+1] == '*' {
						i += 2
						for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
							i++
						}
						if i+1 < n {
							i += 2
						}
						continue
					}
					if src[i] == '{' {
						d++
						i++
						continue
					}
					if src[i] == '}' {
						d--
						i++
						if d == 0 {
							break
						}
						continue
					}
					i++
				}
				end := i
				out = append(out, block{kind: kw, name: name, start: start, braceStart: braceStart, end: end})
				continue
			}
		}
		i++
	}
	return out
}

func isIdentStart(b byte) bool { return (b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')) }
func isIdent(b byte) bool      { return isIdentStart(b) || (b >= '0' && b <= '9') }
func isSpace(b byte) bool      { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

func stripComments(s string) string {
	var b strings.Builder
	n := len(s)
	for i := 0; i < n; {
		if i+1 < n && s[i] == '/' && s[i+1] == '/' {
			for i < n && s[i] != '\n' {
				i++
			}
			b.WriteByte('\n')
			if i < n {
				i++
			}
			continue
		}
		if i+1 < n && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < n && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func extractTypeRefs(s string) []string {
	re := regexp.MustCompile(`(?m)^\s*(?:repeated|optional)?\s*([^\s=]+(?:\s*<[^;>]+>)?)\s+[A-Za-z_][\w]*\s*=\s*\d+`)
	m := re.FindAllStringSubmatch(s, -1)
	var out []string
	for _, g := range m {
		if len(g) < 2 {
			continue
		}
		typ := strings.TrimSpace(g[1])
		if strings.HasPrefix(typ, "map") {
			lt := strings.Index(typ, "<")
			gt := strings.LastIndex(typ, ">")
			if lt >= 0 && gt > lt+1 {
				inside := typ[lt+1 : gt]
				parts := strings.Split(inside, ",")
				if len(parts) == 2 {
					v := strings.TrimSpace(parts[1])
					out = append(out, v)
				}
			}
			continue
		}
		out = append(out, typ)
	}
	return out
}

func extractOriginalBlock(_ []byte, defText string) string { return defText }

func stripSelfPackageQualifiers(content string, selfPkg string) string {
	if strings.TrimSpace(selfPkg) == "" {
		return content
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(selfPkg) + `\.` + `([A-Za-z_][\w]*)`)
	return re.ReplaceAllString(content, `$1`)
}

func transformFieldNames(def string, caseKind string) string {
	i := strings.Index(def, "{")
	j := strings.LastIndex(def, "}")
	if i < 0 || j <= i {
		return def
	}
	head := def[:i+1]
	body := def[i+1 : j]
	tail := def[j:]

	lines := strings.Split(body, "\n")
	fieldRe := regexp.MustCompile(`^([\t ]*(?:repeated[\t ]+)?(?:map\s*<[^>]+>|[^\s=]+)[\t ]+)([A-Za-z_][\w]*)([\t ]*=\s*\d+.*;.*)$`)
	for idx, ln := range lines {
		if m := fieldRe.FindStringSubmatch(ln); m != nil {
			name := m[2]
			switch strings.ToLower(caseKind) {
			case "snake":
				name = toSnake(name)
			case "compact":
				name = strings.ToLower(removeDelims(name))
			default:
				name = snakeToCamel(name)
			}
			lines[idx] = m[1] + name + m[3]
		}
	}
	return head + strings.Join(lines, "\n") + tail
}
