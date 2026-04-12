package log_util

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultRotateMaxBytes = 1 * 1024 * 1024

type rotatingFileWriter struct {
	dir      string
	baseName string
	maxBytes int64
	mu       sync.Mutex
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return 0, err
	}

	latest := filepath.Join(w.dir, w.baseName+"-"+"latest.log")
	if err := w.rotateIfNeeded(latest, int64(len(p))); err != nil {
		return 0, err
	}

	f, err := os.OpenFile(latest, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return f.Write(p)
}

func (w *rotatingFileWriter) rotateIfNeeded(latest string, incoming int64) error {
	if incoming >= w.maxBytes {
		return w.rotate(latest)
	}

	info, err := os.Stat(latest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Size()+incoming <= w.maxBytes {
		return nil
	}

	return w.rotate(latest)
}

func (w *rotatingFileWriter) rotate(latest string) error {
	if _, err := os.Stat(latest); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	ts := time.Now().Format("20060102150405")
	archive := filepath.Join(w.dir, w.baseName+"-"+ts+".log")
	archive = uniqueArchivePath(archive)

	return os.Rename(latest, archive)
}
