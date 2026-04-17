package pruner

import (
	"regexp"
	"strings"
)

// Regex patterns used by pruning helpers.
var (
	reLineComment  = regexp.MustCompile(`(?m)//.*$`)
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reMapType      = regexp.MustCompile(`map\s*<\s*([A-Za-z_][\w\.]*)\s*,\s*([A-Za-z_][\w\.]*)\s*>`)
	reFieldType    = regexp.MustCompile(`(?m)(?:^|[\s{])(?:repeated|optional)?\s*([A-Za-z_][\w\.]*)\s+[A-Za-z_][\w]*\s*=\s*\d+\s*;`)
)

// keepFieldStmt reports whether a field statement should be kept based on keepSet.
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

// lastIdent extracts the last identifier from a string.
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

// looksLikeBlock reports whether the text starts with a block keyword.
func looksLikeBlock(s string) bool {
	s = strings.TrimLeft(s, " \t\r\n")
	return strings.HasPrefix(s, "oneof ") || strings.HasPrefix(s, "message ") || strings.HasPrefix(s, "enum ") || strings.HasPrefix(s, "extend ")
}

// readKeyword reads a keyword starting at position i.
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

// readIdentAfter reads an identifier starting at position i.
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

// findBlock finds the matching brace block starting at position i.
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
