package formatter

import (
	"regexp"
	"strings"

	"github.com/aura-studio/proto-converter/converter/core/model"
)

// StripSelfPackageQualifiers 移除当前包的冗余限定前缀。
func (OutputFormatter) StripSelfPackageQualifiers(content, selfPkg string) string {
	if strings.TrimSpace(selfPkg) == "" {
		return content
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(selfPkg) + `\.` + `([A-Za-z_][\w]*)`)
	return re.ReplaceAllString(content, `$1`)
}

// TransformFieldNames 按指定命名风格转换字段名。
func (OutputFormatter) TransformFieldNames(def, caseKind string) string {
	// keep：保持字段名不变
	if strings.ToLower(strings.TrimSpace(caseKind)) == "keep" {
		return def
	}
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
			lines[idx] = m[1] + model.ToCase(m[2], caseKind) + m[3]
		}
	}
	return head + strings.Join(lines, "\n") + tail
}
