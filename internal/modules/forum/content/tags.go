package content

import (
	"regexp"
	"strings"
)

var tagRe = regexp.MustCompile(`(^|[^\p{L}\p{N}_#])#([\p{L}\p{N}_][\p{L}\p{N}_-]{0,23})`)

// ExtractTags 从正文中提取标签，按文档顺序去重并保留首次出现的写法。
// 代码块、链接、引用与提及 chip 中的内容不参与标签提取。
func ExtractTags(input string) []string {
	return Parse(input).Tags()
}

// NormalizeTag 返回标签的小写归一化形式。
func NormalizeTag(tag string) string {
	return strings.ToLower(tag)
}
