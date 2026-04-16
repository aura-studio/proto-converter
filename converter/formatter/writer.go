package formatter

import "strings"

// WriteLangNamespaceOption 写入语言相关的命名空间 option。
func (OutputFormatter) WriteLangNamespaceOption(b *strings.Builder, lang, ns string) {
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
