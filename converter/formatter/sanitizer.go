package formatter

import (
	"regexp"
	"strings"

	"github.com/aura-studio/proto-converter/converter/model"
)

var reservedLineRe = regexp.MustCompile(`(?i)^\s*reserved\b[^;]*;\s*$`)

// ScannerIface is a local interface for scanning top-level blocks (avoids importing converter).
type ScannerIface interface {
	ScanTopLevelBlocks(src string) []model.Block
}

// OutputFormatter 负责 proto 输出的格式化和清理。
type OutputFormatter struct {
	Parser ScannerIface // 用于 ScanTopLevelBlocks
}

// Sanitize 对 proto 输出执行完整的清理流程。
func (f OutputFormatter) Sanitize(src string) string {
	noCmt := stripCommentsOut(src)
	noRes := dropReservedLines(noCmt)
	// 先全局归一化一次空行
	compact := normalizeBlankLines(noRes)
	// 消除块内（message/enum）字段间空行
	compact = f.dropBlankLinesInsideTopBlocks(compact)
	// 修复花括号附近空行
	compact = tightenBlockBlankLines(compact)
	// 最终再标准化一次，确保没有残留空行问题
	return normalizeBlankLines(compact)
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
	reAfterOpen := regexp.MustCompile(`\{\r?\n(?:[\t ]*\r?\n)+`)
	s = reAfterOpen.ReplaceAllString(s, "{\n")
	reBeforeClose := regexp.MustCompile(`\r?\n(?:[\t ]*\r?\n)+\}`)
	s = reBeforeClose.ReplaceAllString(s, "\n}")
	return s
}

// dropBlankLinesInsideTopBlocks removes blank lines inside top-level message/enum bodies.
func (f OutputFormatter) dropBlankLinesInsideTopBlocks(s string) string {
	blocks := f.Parser.ScanTopLevelBlocks(s)
	if len(blocks) == 0 {
		return s
	}
	// 从后往前替换，避免索引位移
	for i := len(blocks) - 1; i >= 0; i-- {
		b := blocks[i]
		inner := model.BlockBody(s, b)
		lines := strings.Split(inner, "\n")
		var kept []string
		for _, ln := range lines {
			if strings.TrimSpace(ln) == "" {
				continue
			}
			kept = append(kept, ln)
		}
		// 保持花括号内首尾各一行换行，主体去除空行
		rebuilt := "\n" + strings.Join(kept, "\n") + "\n"
		s = s[:b.BraceStart+1] + rebuilt + s[b.End-1:]
	}
	return s
}
