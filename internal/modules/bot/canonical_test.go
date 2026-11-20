package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCanonicalFormat(t *testing.T) {
	got := Canonical("GET", "/bot-v1/ping", "default", "1700000000", "abcdefghijklmnop", nil)
	want := strings.Join([]string{
		"v1",
		"default",
		"GET",
		"/bot-v1/ping",
		"1700000000",
		"abcdefghijklmnop",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}, "\n")

	if got != want {
		t.Fatalf("规范化字符串不正确:\n得到: %q\n期望: %q", got, want)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatal("规范化字符串不应以换行结尾")
	}
}

func TestCanonicalIncludesBody(t *testing.T) {
	body := []byte(`{"message":"hi"}`)
	sum := sha256.Sum256(body)
	got := Canonical("POST", "/bot-v1/echo", "default", "1700000000", "abcdefghijklmnop", body)

	if !strings.HasSuffix(got, "\n"+hex.EncodeToString(sum[:])) {
		t.Fatalf("body 未参与规范化: %q", got)
	}
	emptySum := sha256.Sum256(nil)
	if strings.HasSuffix(got, "\n"+hex.EncodeToString(emptySum[:])) {
		t.Fatalf("body 被当成了空请求体: %q", got)
	}
}

func TestCanonicalChangesWithEveryField(t *testing.T) {
	base := Canonical("GET", "/bot-v1/ping", "default", "1700000000", "abcdefghijklmnop", nil)

	cases := []struct {
		name string
		got  string
	}{
		{"method", Canonical("POST", "/bot-v1/ping", "default", "1700000000", "abcdefghijklmnop", nil)},
		{"uri", Canonical("GET", "/bot-v1/ping?x=1", "default", "1700000000", "abcdefghijklmnop", nil)},
		{"key_id", Canonical("GET", "/bot-v1/ping", "other", "1700000000", "abcdefghijklmnop", nil)},
		{"timestamp", Canonical("GET", "/bot-v1/ping", "default", "1700000001", "abcdefghijklmnop", nil)},
		{"nonce", Canonical("GET", "/bot-v1/ping", "default", "1700000000", "abcdefghijklmnoq", nil)},
		{"body", Canonical("GET", "/bot-v1/ping", "default", "1700000000", "abcdefghijklmnop", []byte("x"))},
	}

	for _, tc := range cases {
		if tc.got == base {
			t.Errorf("%s 变化后规范化字符串未改变", tc.name)
		}
	}
}
