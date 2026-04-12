package log_util

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColorizePrefix(t *testing.T) {
	got := colorizePrefix(InfoLevel, colorGreen)
	want := colorGreen + "[INFO]" + colorReset + " "
	if got != want {
		t.Fatalf("unexpected colorized prefix, got %q want %q", got, want)
	}
}

func TestRotatingFileWriterRotateBySize(t *testing.T) {
	dir := t.TempDir()
	w := &rotatingFileWriter{
		dir:      dir,
		baseName: "monitor-server",
		maxBytes: 64,
	}

	first := []byte(strings.Repeat("a", 40))
	second := []byte(strings.Repeat("b", 40))

	if _, err := w.Write(first); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if _, err := w.Write(second); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	latest := filepath.Join(dir, "monitor-server-latest.log")
	latestContent, err := os.ReadFile(latest)
	if err != nil {
		t.Fatalf("read latest failed: %v", err)
	}
	if string(latestContent) != string(second) {
		t.Fatalf("latest content mismatch, got %q", string(latestContent))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}

	hasArchive := false
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "monitor-server-") && strings.HasSuffix(name, ".log") && name != "monitor-server-latest.log" {
			hasArchive = true
			break
		}
	}

	if !hasArchive {
		t.Fatalf("expected at least one rotated archive file in %s", dir)
	}
}

func TestAppendFileWriterNoRotate(t *testing.T) {
	dir := t.TempDir()
	w := &appendFileWriter{dir: dir, filename: "main.log"}

	if _, err := w.Write([]byte(strings.Repeat("x", 1024))); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if _, err := w.Write([]byte(strings.Repeat("y", 1024))); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "main.log" {
		t.Fatalf("expected only main.log, got entries: %v", entries)
	}
}

func TestPrefixedWriterFileHasNoAnsi(t *testing.T) {
	dir := t.TempDir()
	fileOut := &rotatingFileWriter{
		dir:      dir,
		baseName: "monitor-file-sync",
		maxBytes: 1024 * 1024,
	}

	logger := newDoubleLogger(bytes.NewBuffer(nil), fileOut, InfoLevel, colorBlue)
	logger.Printf("test message")

	data, err := os.ReadFile(filepath.Join(dir, "monitor-file-sync-latest.log"))
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}

	if bytes.Contains(data, []byte("\x1b[")) {
		t.Fatalf("file log should not contain ANSI color codes: %q", string(data))
	}
}
