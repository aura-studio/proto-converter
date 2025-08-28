package converter

import "strings"

func toCamel(s string) string {
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
	parts := strings.FieldsFunc(s, isDelim)
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

func toSnake(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var result []rune
	prevUnderscore := false
	runes := []rune(s)
	for i, r := range runes {
		if isDelim(r) {
			if !prevUnderscore {
				result = append(result, '_')
				prevUnderscore = true
			}
			continue
		}

		if isUpper(r) {
			// 只要不是第一个字符且前一个不是下划线，就加下划线
			if i > 0 && !prevUnderscore {
				result = append(result, '_')
			}
			result = append(result, toLower(r))
			prevUnderscore = false
		} else {
			result = append(result, r)
			prevUnderscore = false
		}
	}
	res := string(result)
	// 合并多余下划线
	for strings.Contains(res, "__") {
		res = strings.ReplaceAll(res, "__", "_")
	}
	// 去除首尾下划线
	res = strings.Trim(res, "_")
	return res
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
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

func toLower(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

func isDelim(r rune) bool {
	return r == '_' || r == '-' || r == '.' || r == ' '
}

func removeDelims(s string) string {
	b := strings.Builder{}
	for _, r := range s {
		if isDelim(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func toCase(s, caseKind string) string {
	switch strings.ToLower(strings.TrimSpace(caseKind)) {
	case "camel":
		return toCamel(s)
	case "snake":
		return toSnake(s)
	case "compact":
		return strings.ToLower(removeDelims(s))
	case "keep":
		fallthrough
	default:
		return s
	}
}
