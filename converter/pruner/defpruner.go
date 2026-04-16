package pruner

import "strings"

// DefinitionPruner handles message/enum definition-level field pruning.
type DefinitionPruner struct{}

// PruneMessageFields prunes message fields based on keepSet.
func (DefinitionPruner) PruneMessageFields(def string, keepSet map[string]struct{}) string {
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

// pruneOneofFields prunes oneof block fields based on keepSet.
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

// PruneOneofFields prunes oneof block fields based on keepSet (exported method).
func (DefinitionPruner) PruneOneofFields(blk string, keepSet map[string]struct{}) string {
	return pruneOneofFields(blk, keepSet)
}

// CollectTypeTokens collects all type tokens from a definition text.
func (DefinitionPruner) CollectTypeTokens(def string) []string {
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

// ResolveTypeKeepSet looks up the keepSet for a type by name and package.
func ResolveTypeKeepSet(m map[string]map[string]struct{}, pkg, name string) map[string]struct{} {
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

// ExtractOriginalBlock extracts the original block text from source data.
func ExtractOriginalBlock(_ []byte, defText string) string { return defText }
