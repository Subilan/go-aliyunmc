package content

import (
	"strings"
	"testing"
)

func TestSanitizeStripsDangerousMarkup(t *testing.T) {
	input := `<p onclick="x()">hi<script>alert(1)</script><img src="javascript:alert(1)" onerror="x()"><a href="javascript:alert(1)">bad</a></p>`
	out, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	for _, bad := range []string{"script", "onclick", "onerror", "javascript:"} {
		if strings.Contains(strings.ToLower(out), bad) {
			t.Fatalf("output still contains %q: %s", bad, out)
		}
	}
}

func TestSanitizeRejectsDataImage(t *testing.T) {
	if _, err := Sanitize(`<p><img src="data:image/png;base64,AAAA"></p>`); err != ErrDataImage {
		t.Fatalf("want ErrDataImage, got %v", err)
	}
	// 属性值中的 `>` 不应让检测失效。
	if _, err := Sanitize(`<p><img alt=">" src="data:image/png;base64,AAAA"></p>`); err != ErrDataImage {
		t.Fatalf("quoted > bypass: want ErrDataImage, got %v", err)
	}
}

func TestSanitizeKeepsTiptapStructure(t *testing.T) {
	input := `<h2>标题</h2><p><strong>粗体</strong><em>斜体</em></p><ul><li>一</li></ul><pre><code class="language-go">fmt.Println()</code></pre><blockquote><p>引用</p></blockquote><table><tbody><tr><th colspan="2">h</th></tr><tr><td style="text-align:center">c</td></tr></tbody></table><p><img src="/forum-v1/files/2026/11/0123456789abcdef0123456789abcdef.webp" alt="a" width="10" height="20" data-align="center"></p><p><span class="reply-reference" data-tag-type="reply" data-floor="3">#3</span> <span class="mention" data-tag-type="mention" data-user-id="7">@bob</span></p>`
	out, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	for _, want := range []string{"<h2>", "<strong>", "<em>", "<ul>", "<li>", `class="language-go"`, "<blockquote>", "<table>", `colspan="2"`, "text-align:center", `data-align="center"`, `data-tag-type="reply"`, `data-floor="3"`, `data-tag-type="mention"`, `data-user-id="7"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q: %s", want, out)
		}
	}
}

func TestPlainTextAndExcerpt(t *testing.T) {
	text := PlainText(`<h1>标题</h1><p>第一段<br>换行</p><p><img src="/forum-v1/files/2026/11/0123456789abcdef0123456789abcdef.png" alt="图片"></p><script>bad()</script>`)
	if strings.Contains(text, "bad") {
		t.Fatalf("script leaked into text: %q", text)
	}
	for _, want := range []string{"标题", "第一段", "换行", "图片"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q: %q", want, text)
		}
	}
	if strings.Contains(text, "  ") {
		t.Fatalf("whitespace not collapsed: %q", text)
	}

	if got := Excerpt("一二三四五", 3); got != "一二三…" {
		t.Fatalf("Excerpt = %q", got)
	}
	if got := Excerpt("一二三", 3); got != "一二三" {
		t.Fatalf("Excerpt = %q", got)
	}
}

func TestExtractTags(t *testing.T) {
	input := `<p>#初见 #Go-lang #初见</p><pre>#code</pre><a href="#">#link</a><span data-tag-type="mention" data-user-id="1">#提及</span><p>#中文 标签#不是</p>`
	tags := ExtractTags(input)
	want := []string{"初见", "Go-lang", "中文"}
	if len(tags) != len(want) {
		t.Fatalf("tags = %v, want %v", tags, want)
	}
	for i := range want {
		if tags[i] != want[i] {
			t.Fatalf("tags = %v, want %v", tags, want)
		}
	}
}

func TestMentions(t *testing.T) {
	html := `<p><span class="mention" data-tag-type="mention" data-user-id="7">@old</span><span class="whatever" data-tag-type="mention" data-user-id="8">@x</span><span class="mention" data-tag-type="mention" data-user-id="99">@gone</span></p>`
	ids := MentionIDs(html)
	if len(ids) != 3 || ids[0] != 7 || ids[1] != 8 || ids[2] != 99 {
		t.Fatalf("MentionIDs = %v", ids)
	}

	out, validIDs := NormalizeMentions(html, map[uint]string{7: "alice", 8: "bob"})
	if len(validIDs) != 2 || validIDs[0] != 7 || validIDs[1] != 8 {
		t.Fatalf("valid mention ids = %v", validIDs)
	}
	if !strings.Contains(out, `data-user-id="7"`) || !strings.Contains(out, ">@alice<") {
		t.Fatalf("valid mention not rebuilt: %s", out)
	}
	if !strings.Contains(out, `data-user-id="8"`) || !strings.Contains(out, ">@bob<") {
		t.Fatalf("valid mention not rebuilt: %s", out)
	}
	if strings.Contains(out, `data-user-id="99"`) {
		t.Fatalf("invalid mention should be plain text: %s", out)
	}

	// 无效 chip 内部嵌套的有效 chip 也应被规范化。
	nested := `<p><span data-tag-type="mention" data-user-id="99"><span data-tag-type="mention" data-user-id="7">@old</span></span></p>`
	out, ids = NormalizeMentions(nested, map[uint]string{7: "alice"})
	if len(ids) != 1 || ids[0] != 7 || !strings.Contains(out, ">@alice<") || strings.Contains(out, `data-user-id="99"`) {
		t.Fatalf("nested mention normalization = %q, ids=%v", out, ids)
	}
}

func TestReplyFloors(t *testing.T) {
	html := `<p><span class="reply-reference" data-tag-type="reply" data-floor="3">#3</span></p><p><span data-tag-type="reply" data-floor="9">#9</span></p>`
	floors := ReplyFloors(html)
	if len(floors) != 2 || floors[0] != 3 || floors[1] != 9 {
		t.Fatalf("floors = %v", floors)
	}
}

func TestValidateTitle(t *testing.T) {
	if _, err := ValidateTitle("   "); err == nil {
		t.Fatal("empty title should fail")
	}
	if _, err := ValidateTitle("a\nb"); err == nil {
		t.Fatal("newline title should fail")
	}
	got, err := ValidateTitle("  你好  ")
	if err != nil || got != "你好" {
		t.Fatalf("got %q, %v", got, err)
	}
}
