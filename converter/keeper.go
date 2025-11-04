package converter

import (
	"log"
	"path/filepath"
	"strings"
)

type FileKeeperItem struct {
	OsPath string
	Path   string
	Dir    string
	Base   string
	Keep   map[string]bool
}

type FileKeeper struct {
	Files map[string]FileKeeperItem
}

func NewFileKeeper(c Config) FileKeeper {
	f := FileKeeper{}
	f.Files = map[string]FileKeeperItem{}
	for _, fr := range c.Import.Keep.Files {
		t := strings.TrimSpace(fr.File)
		if t == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(t), ".proto") {
			t += ".proto"
		}
		ni, err := normalizeItem(t)
		if err != nil {
			log.Panicln(err)
		}

		var item FileKeeperItem
		item.OsPath = filepath.Join(c.Import.Dir, ni.Path)
		item.Path = ni.Path
		item.Dir = ni.Dir
		item.Base = ni.Base
		item.Keep = map[string]bool{}

		for _, k := range fr.Keep {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			item.Keep[k] = true
		}

		f.Files[item.OsPath] = item
	}

	return f
}

type TypeKeeper struct {
	Types map[string]map[string]bool
}

func NewTypeKeeper(c Config) TypeKeeper {
	t := TypeKeeper{}
	t.Types = map[string]map[string]bool{}
	for _, tr := range c.Import.Keep.Types {
		tname := strings.TrimSpace(tr.Type)
		if tname == "" {
			continue
		}
		set := map[string]bool{}
		for _, f := range tr.Keep {
			f = strings.TrimSpace(f)
			if f != "" {
				set[f] = true
			}
		}
		t.Types[tname] = set
	}

	return t
}
