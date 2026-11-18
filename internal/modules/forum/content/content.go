// Package content 是论坛正文的唯一可信处理管线。
// 所有主题与跟帖的 HTML 都必须先经过这里，再写入数据库。
package content

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// ErrDataImage 表示正文包含 data: 图片。正文图片必须先上传为站内 URL。
var ErrDataImage = errors.New("正文不能包含 data: 图片，请先上传图片")

var (
	imgSrcRe = regexp.MustCompile(`(?i)^(?:https?://[^\s]+|/forum-v1/files/[0-9]{4}/[0-9]{2}/[a-f0-9]{32}\.(?:jpg|png|webp))$`)
	hrefRe   = regexp.MustCompile(`(?i)^(?:https?://[^\s]+|mailto:[^\s]+|/[^\s]*)$`)

	codeClassRe = regexp.MustCompile(`^language-[a-z0-9-]+$`)
	spanClassRe = regexp.MustCompile(`^(?:reply-reference|mention)$`)
	tagTypeRe   = regexp.MustCompile(`^(?:reply|mention)$`)
	digitsRe    = regexp.MustCompile(`^\d+$`)
	alignRe     = regexp.MustCompile(`^(?:left|center|right)$`)

	tableStyleRe = regexp.MustCompile(`^(?:(?:width|min-width):\s*\d+px;?\s*)+$`)
	colStyleRe   = regexp.MustCompile(`^width:\s*\d+px;?$`)
	cellStyleRe  = regexp.MustCompile(`^text-align:\s*(?:left|center|right);?$`)
)

var sanitizer = newSanitizer()

func newSanitizer() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u", "s", "del", "code", "pre", "blockquote",
		"ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "a", "img",
		"table", "thead", "tbody", "tfoot", "tr", "th", "td", "colgroup", "col",
		"div", "span",
	)

	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(true)

	// 链接：http/https/mailto 与站内相对路径。
	p.AllowAttrs("href").Matching(hrefRe).OnElements("a")
	p.AllowAttrs("title").OnElements("a")

	// 图片：只允许已上传的站内文件或外链 http(s) 地址。
	p.AllowAttrs("src").Matching(imgSrcRe).OnElements("img")
	p.AllowAttrs("alt").OnElements("img")
	p.AllowAttrs("width", "height").Matching(digitsRe).OnElements("img")
	p.AllowAttrs("data-align").Matching(alignRe).OnElements("img")

	// 代码块语言标记。
	p.AllowAttrs("class").Matching(codeClassRe).OnElements("code")

	// 表格包装容器。
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^tableWrapper$`)).OnElements("div")

	// 楼层引用与提及 chip。class 只用于样式，服务端以 data-tag-type 识别。
	p.AllowAttrs("class").Matching(spanClassRe).OnElements("span")
	p.AllowAttrs("data-tag-type").Matching(tagTypeRe).OnElements("span")
	p.AllowAttrs("data-floor").Matching(digitsRe).OnElements("span")
	p.AllowAttrs("data-user-id").Matching(digitsRe).OnElements("span")

	// 表格与单元格尺寸/对齐。
	p.AllowAttrs("style").Matching(tableStyleRe).OnElements("table")
	p.AllowAttrs("style").Matching(colStyleRe).OnElements("col")
	p.AllowAttrs("colspan", "rowspan", "colwidth").Matching(digitsRe).OnElements("th", "td")
	p.AllowAttrs("style").Matching(cellStyleRe).OnElements("th", "td")

	return p
}

// Sanitize 按论坛白名单清洗正文 HTML。
// data: 图片在清洗前就会被拒绝，避免体积失控与无法审计的嵌入内容。
func Sanitize(input string) (string, error) {
	if hasDataImage(input) {
		return "", ErrDataImage
	}

	cleaned := sanitizer.Sanitize(input)
	return strings.TrimSpace(cleaned), nil
}

// hasDataImage 用 HTML 解析而不是正则判断 img[src]，
// 避免属性值中的 `>`（如 alt=">"）让正则漏检。
func hasDataImage(input string) bool {
	doc := parseFragment(input)
	if doc == nil {
		return false
	}

	found := false
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if found {
			return
		}
		if n.Type == html.ElementNode && n.Data == "img" {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(attr(n, "src"))), "data:") {
				found = true
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)

	return found
}

// ValidateTitle 规范化并校验主题标题。
func ValidateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)

	count := 0
	for _, r := range title {
		if r == '\n' || r == '\r' || r == '\t' || r < 0x20 || r == 0x7f {
			return "", errors.New("标题不能包含换行或控制字符")
		}
		count++
	}

	if count == 0 {
		return "", errors.New("标题不能为空")
	}
	if count > 120 {
		return "", fmt.Errorf("标题不能超过 120 个字符（当前 %d）", count)
	}

	return title, nil
}
