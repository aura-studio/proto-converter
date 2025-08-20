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
	b := strings.Builder{}
	upperNext := true
	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			upperNext = true
			continue
		}
		if upperNext {
			if 'a' <= r && r <= 'z' {
				r = r - ('a' - 'A')
			}
			upperNext = false
		}
		b.WriteRune(r)
	}
	return b.String()
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
