package log_util

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestLogger(t *testing.T, dir string, logName string, maxSizeMB int) (*Logger, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	logger := newLogger(loggerOptions{
		logPath:    filepath.Join(dir, logName),
		maxSizeMB:  maxSizeMB,
		console:    out,
		consoleErr: errOut,
		color:      true,
	})
	return logger, out, errOut
}

func TestLoggerFileHasNoAnsi(t *testing.T) {
	dir := t.TempDir()
	logger, out, _ := newTestLogger(t, dir, "monitor-file-sync-latest.log", defaultRotateMaxMB)

	logger.Info("test message")

	file, err := os.ReadFile(filepath.Join(dir, "monitor-file-sync-latest.log"))
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if bytes.Contains(file, []byte("\x1b[")) {
		t.Fatalf("file log should not contain ANSI color codes: %q", string(file))
	}
	if !strings.Contains(string(file), "test message") {
		t.Fatalf("file log should contain the message, got %q", string(file))
	}
	if !bytes.Contains(out.Bytes(), []byte("\x1b[")) {
		t.Fatalf("console log should be colorized, got %q", out.String())
	}
}

func TestLoggerRotatesBySize(t *testing.T) {
	dir := t.TempDir()
	logger, _, _ := newTestLogger(t, dir, "monitor-server-latest.log", 1)

	payload := strings.Repeat("a", 700*1024)
	logger.Info("first")
	logger.Info("%s", payload)
	logger.Info("%s", payload)

	if _, err := os.Stat(filepath.Join(dir, "monitor-server-latest.log")); err != nil {
		t.Fatalf("latest log missing: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}

	hasArchive := false
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "monitor-server-latest-") && strings.HasSuffix(name, ".log") {
			hasArchive = true
			break
		}
	}
	if !hasArchive {
		t.Fatalf("expected at least one rotated archive file in %s", dir)
	}
}

func TestFatalGoesToStderrAndFile(t *testing.T) {
	dir := t.TempDir()
	logger, out, errOut := newTestLogger(t, dir, "fatal-latest.log", defaultRotateMaxMB)

	logger.log(LevelFatal, "boom")

	if out.Len() != 0 {
		t.Fatalf("fatal should not write to stdout, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), "FATAL") || !strings.Contains(errOut.String(), "boom") {
		t.Fatalf("unexpected stderr output %q", errOut.String())
	}

	file, err := os.ReadFile(filepath.Join(dir, "fatal-latest.log"))
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if !strings.Contains(string(file), "level=FATAL") {
		t.Fatalf("file log should carry FATAL level, got %q", string(file))
	}
}

func TestFormatMessage(t *testing.T) {
	if got := formatMessage("value=%d", 1); got != "value=1" {
		t.Fatalf("unexpected formatted message %q", got)
	}
	if got := formatMessage("plain message"); got != "plain message" {
		t.Fatalf("unexpected plain message %q", got)
	}
}

func TestSanitizeBaseName(t *testing.T) {
	if got := sanitizeBaseName("[monitor] file/sync"); got != "monitor_file_sync" {
		t.Fatalf("unexpected sanitized name %q", got)
	}
	if got := sanitizeBaseName("   "); got != "prefixed_" {
		t.Fatalf("unexpected fallback name %q", got)
	}
}

func TestResolveDefaultLogsDirUsesEnv(t *testing.T) {
	t.Setenv("GO_ALIYUNMC_LOG_DIR", "/tmp/aliyunmc-custom-logs")
	if got := resolveDefaultLogsDir(); got != "/tmp/aliyunmc-custom-logs" {
		t.Fatalf("unexpected logs dir %q", got)
	}
}
