package str_util

import (
	"testing"
	"unicode/utf8"
)

func TestTruncateKeepsUTF8Boundary(t *testing.T) {
	// "你" 占 3 字节；上限 4 字节时只能保留第一个完整字符。
	got := Truncate("你好", 4)
	if got != "你" {
		t.Fatalf("Truncate = %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("invalid utf8: %q", got)
	}

	if got := Truncate("abc", 3); got != "abc" {
		t.Fatalf("no-op truncate = %q", got)
	}
	if got := Truncate("你好", 10); got != "你好" {
		t.Fatalf("short string = %q", got)
	}
	if got := Truncate("你好", 0); got != "你好" {
		t.Fatalf("zero limit = %q", got)
	}
}
