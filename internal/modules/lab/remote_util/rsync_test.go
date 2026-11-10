package remote_util

import (
	"strings"
	"testing"
)

func TestRsyncOutputWriterSplitsLines(t *testing.T) {
	var lines []string
	w := &rsyncOutputWriter{tailLimit: 3, onLine: func(line string) { lines = append(lines, line) }}

	if _, err := w.Write([]byte("sending incremental file list\ncert.pem\n")); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	if _, err := w.Write([]byte("sent 100 bytes")); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	w.flush()

	want := []string{"sending incremental file list", "cert.pem", "sent 100 bytes"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("回调输出不符: %v", lines)
	}
}

func TestRsyncOutputWriterKeepsTail(t *testing.T) {
	w := &rsyncOutputWriter{tailLimit: 2}
	for _, line := range []string{"a", "b", "c", "d"} {
		if _, err := w.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("写入失败: %v", err)
		}
	}

	if got := w.detail(); got != "c\nd" {
		t.Fatalf("诊断信息不符: %q", got)
	}
}

func TestRsyncOutputWriterToleratesNilOnLine(t *testing.T) {
	w := &rsyncOutputWriter{tailLimit: 5}
	if _, err := w.Write([]byte("hello\nworld")); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	w.flush()

	if got := w.detail(); got != "hello\nworld" {
		t.Fatalf("诊断信息不符: %q", got)
	}
}
