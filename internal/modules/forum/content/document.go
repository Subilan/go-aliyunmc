package content

import (
	"bytes"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var blockElements = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true, "br": true,
	"div": true, "dl": true, "dt": true, "dd": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "header": true, "hr": true, "li": true,
	"main": true, "nav": true, "ol": true, "p": true, "pre": true, "section": true,
	"table": true, "tbody": true, "td": true, "tfoot": true, "th": true, "thead": true,
	"tr": true, "ul": true,
}

// Document 是一次解析后的 HTML fragment。
// 同一篇正文的纯文本、图片数、标签、楼层引用与提及都基于它计算，避免重复解析。
type Document struct {
	source string
	root   *html.Node
}

// Parse 解析 HTML fragment。解析失败时返回一个空文档。
func Parse(input string) *Document {
	return &Document{source: input, root: parseFragment(input)}
}

// PlainText 提取纯文本，用于搜索、摘要与通知。
func (d *Document) PlainText() string {
	if d.root == nil {
		return ""
	}

	var b strings.Builder
	walkText(d.root, &b)

	return strings.Join(strings.Fields(b.String()), " ")
}

// CountImages 统计图片数量。
func (d *Document) CountImages() int {
	count := 0
	d.visit(func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			count++
		}
	})
	return count
}

// HasExpandableContent 判断是否包含图片或表格，用于放宽“空正文”校验。
func (d *Document) HasExpandableContent() bool {
	found := false
	d.visit(func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "img" || n.Data == "table") {
			found = true
		}
	})
	return found
}

// Tags 从正文中提取标签，按文档顺序去重并保留首次出现的写法。
// 代码块、链接、引用与提及 chip 中的内容不参与标签提取。
func (d *Document) Tags() []string {
	if d.root == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, 4)

	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "code", "pre", "a", "script", "style", "template":
				return
			}
			if hasTagType(n, "reply") || hasTagType(n, "mention") {
				return
			}
		}

		if n.Type == html.TextNode {
			for _, match := range tagRe.FindAllStringSubmatch(n.Data, -1) {
				tag := match[2]
				key := strings.ToLower(tag)
				if !seen[key] {
					seen[key] = true
					result = append(result, tag)
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(d.root)

	return result
}

// Floors 返回正文中楼层引用 chip 的楼层号，按文档顺序。
// 只识别 data-tag-type="reply" 的 span。
func (d *Document) Floors() []uint {
	floors := make([]uint, 0, 2)
	d.visit(func(n *html.Node) {
		if hasTagType(n, "reply") {
			if floor, ok := parseUintAttr(n, "data-floor"); ok {
				floors = append(floors, floor)
			}
		}
	})
	return floors
}

// MentionIDs 返回正文中提及 chip 指向的用户 ID（按文档顺序去重）。
// 只识别 data-tag-type="mention" 的 span，class 不参与判断。
func (d *Document) MentionIDs() []uint {
	seen := make(map[uint]bool)
	ids := make([]uint, 0, 4)

	d.visit(func(n *html.Node) {
		if !hasTagType(n, "mention") {
			return
		}
		if id, ok := parseUintAttr(n, "data-user-id"); ok && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	})
	return ids
}

// NormalizeMentions 依据有效的用户集合重建提及 chip，并返回有效提及的用户 ID。
//
//   - 有效提及：属性收敛为 class="mention"、data-tag-type="mention"、data-user-id，
//     文本重建为 "@username"，避免 chip 文本与目标用户不一致。
//   - 无效提及：降级为纯文本，保留原有文本内容。
//
// 没有提及节点时原样返回 source，避免无意义的重渲染。
func (d *Document) NormalizeMentions(usernames map[uint]string) (string, []uint) {
	if d.root == nil {
		return d.source, []uint{}
	}

	seen := make(map[uint]bool)
	ids := make([]uint, 0, 4)
	changed := false

	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if hasTagType(n, "mention") {
			changed = true
			id, ok := parseUintAttr(n, "data-user-id")
			username, found := usernames[id]
			if !ok || !found {
				// 去掉无效 chip 的外壳，但继续规范化其内部可能嵌套的有效 chip。
				children := make([]*html.Node, 0, 2)
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					children = append(children, child)
				}
				unwrap(n)
				for _, child := range children {
					visit(child)
				}
				return
			}

			n.Attr = []html.Attribute{
				{Key: "class", Val: "mention"},
				{Key: "data-tag-type", Val: "mention"},
				{Key: "data-user-id", Val: strconv.FormatUint(uint64(id), 10)},
			}
			clearChildren(n)
			n.AppendChild(&html.Node{Type: html.TextNode, Data: "@" + username})

			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
			return
		}

		for child := n.FirstChild; child != nil; {
			next := child.NextSibling
			visit(child)
			child = next
		}
	}
	visit(d.root)

	if !changed {
		return d.source, ids
	}
	return renderChildren(d.root), ids
}

// visit 按文档顺序遍历所有节点。
func (d *Document) visit(fn func(*html.Node)) {
	if d.root == nil {
		return
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		fn(n)
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(d.root)
}

// walkText 递归提取文本，块级元素前后补空格，图片取 alt。
func walkText(n *html.Node, b *strings.Builder) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(n.Data)
		return
	case html.ElementNode:
		switch n.Data {
		case "script", "style", "template":
			return
		case "img":
			b.WriteByte(' ')
			b.WriteString(attr(n, "alt"))
			b.WriteByte(' ')
			return
		}

		if blockElements[n.Data] {
			b.WriteByte(' ')
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walkText(child, b)
		}
		if blockElements[n.Data] {
			b.WriteByte(' ')
		}
		return
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		walkText(child, b)
	}
}

// parseFragment 把 HTML 文本解析成挂载在 div 下的节点树。
func parseFragment(input string) *html.Node {
	context := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(input), context)
	if err != nil {
		return nil
	}

	root := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, node := range nodes {
		root.AppendChild(node)
	}

	return root
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func hasTagType(n *html.Node, typ string) bool {
	return n.Type == html.ElementNode && n.Data == "span" && attr(n, "data-tag-type") == typ
}

func parseUintAttr(n *html.Node, key string) (uint, bool) {
	value := strings.TrimSpace(attr(n, key))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(parsed), true
}

// unwrap 用节点的子节点替换节点本身。
func unwrap(n *html.Node) {
	if n.Parent == nil {
		return
	}
	parent := n.Parent
	for n.FirstChild != nil {
		child := n.FirstChild
		n.RemoveChild(child)
		parent.InsertBefore(child, n)
	}
	parent.RemoveChild(n)
}

func clearChildren(n *html.Node) {
	for n.FirstChild != nil {
		n.RemoveChild(n.FirstChild)
	}
}

func renderChildren(root *html.Node) string {
	var buf bytes.Buffer
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&buf, child); err != nil {
			return ""
		}
	}
	return buf.String()
}
