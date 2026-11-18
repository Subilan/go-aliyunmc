package content

// PlainText 从 HTML fragment 中提取纯文本，用于搜索、摘要与通知。
func PlainText(input string) string {
	return Parse(input).PlainText()
}

// Excerpt 截取前 n 个 rune，超出时追加省略号。
func Excerpt(text string, n int) string {
	if n <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= n {
		return text
	}

	return string(runes[:n]) + "…"
}
