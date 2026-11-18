// Package str_util 提供字符串处理的小工具。
package str_util

import "unicode/utf8"

// Truncate 按字节上限截断字符串，并保证不会切断 UTF-8 字符。
func Truncate(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}

	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
