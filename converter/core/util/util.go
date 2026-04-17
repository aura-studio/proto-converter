package util

import "strings"

// BaseName 从可能带包名限定的类型标记中提取简单名称。
// 例如："google.protobuf.Timestamp" → "Timestamp"，"MyMessage" → "MyMessage"
func BaseName(tok string) string {
	t := strings.TrimPrefix(strings.TrimSpace(tok), ".")
	if t == "" {
		return t
	}
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}
