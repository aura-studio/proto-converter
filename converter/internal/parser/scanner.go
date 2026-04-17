package parser

import (
	"regexp"
	"strings"

	"github.com/aura-studio/proto-converter/converter/core/model"
)

// ScanTopLevelBlocks scans proto source for all top-level message/enum definition blocks.
func (ProtoParser) ScanTopLevelBlocks(src string) []model.Block {
	var out []model.Block
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
		if depth == 0 && IsIdentStart(src[i]) {
			start := i
			for i < n && IsIdent(src[i]) {
				i++
			}
			kw := src[start:i]
			if kw == "message" || kw == "enum" {
				for i < n && IsSpace(src[i]) {
					i++
				}
				nameStart := i
				for i < n && IsIdent(src[i]) {
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
				out = append(out, model.Block{Kind: kw, Name: name, Start: start, BraceStart: braceStart, End: end})
				continue
			}
		}
		i++
	}
	return out
}

// StripComments removes line comments (//) and block comments (/* */) from proto source.
func (ProtoParser) StripComments(s string) string {
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

// ExtractTypeRefs extracts type references from comment-stripped proto source.
func (ProtoParser) ExtractTypeRefs(s string) []string {
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
