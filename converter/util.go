package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type protoItem struct {
	Path string
	Dir  string
	Base string
}

func ensureDir(dir string, dry bool) error {
	if dry {
		fmt.Printf("[dry] mkdir -p %s\n", dir)
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func snakeToCamel(s string) string {
	s = strings.TrimSpace(s)
	// 全大写直接返回
	if s == strings.ToUpper(s) {
		return s
	}
	// 首字母大写且后续全大写也直接返回（如 ASNBe）
	if len(s) > 1 && isUpper(rune(s[0])) && s[1:] == strings.ToUpper(s[1:]) {
		return s
	}
	// 分隔符分段，每段首字母大写
	parts := splitByDelims(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		if isLower(runes[0]) {
			runes[0] = toUpper(runes[0])
		}
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

func splitByDelims(s string) []string {
	f := func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ' '
	}
	return strings.FieldsFunc(s, f)
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}
func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}
func toUpper(r rune) rune {
	if isLower(r) {
		return r - ('a' - 'A')
	}
	return r
}

func normalizeItem(s string) (protoItem, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "\\")
	if s == "" {
		return protoItem{}, fmt.Errorf("空的 proto 条目")
	}
	base := filepath.Base(s)
	dir := filepath.Dir(s)
	if dir == "." {
		dir = ""
	}
	return protoItem{Path: s, Dir: dir, Base: base}, nil
}
func exists(p string) bool      { _, err := os.Stat(p); return err == nil }
func shortPath(p string) string { return filepath.ToSlash(p) }

func trimExt(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

var importRe = regexp.MustCompile(`(?m)^\s*import\s+\"([^\"]+)\"\s*;`)
