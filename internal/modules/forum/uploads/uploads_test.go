package uploads_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/uploads"

	"github.com/creasty/defaults"
)

func testConfig(t *testing.T, dir string) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	if err := defaults.Set(cfg); err != nil {
		t.Fatalf("defaults: %v", err)
	}
	cfg.Forum.UploadDir = dir
	config.Set(cfg)
	return cfg
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 1, G: 2, B: 3, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestSaveAndDelete(t *testing.T) {
	dir := t.TempDir()
	testConfig(t, dir)

	saved, err := uploads.Save(pngBytes(t, 4, 4), forummodels.UploadKindPost)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := uploads.Path(saved.Key); err != nil {
		t.Fatalf("key %q rejected: %v", saved.Key, err)
	}
	if saved.Mime != "image/png" || saved.Width != 4 || saved.Height != 4 {
		t.Fatalf("saved = %+v", saved)
	}
	if len(saved.SHA256) != 64 {
		t.Fatalf("sha256 = %q", saved.SHA256)
	}

	path, err := uploads.Path(saved.Key)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}

	if !strings.HasPrefix(uploads.URL(saved.Key), uploads.URLPrefix) {
		t.Fatalf("url = %q", uploads.URL(saved.Key))
	}

	if err := uploads.Delete(saved.Key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
}

func TestRejectsUnsupportedAndInvalid(t *testing.T) {
	testConfig(t, t.TempDir())

	if _, err := uploads.Save([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"), forummodels.UploadKindPost); err != uploads.ErrUnsupportedType {
		t.Fatalf("gif err = %v", err)
	}

	if _, err := uploads.Save([]byte("not an image"), forummodels.UploadKindPost); err != uploads.ErrUnsupportedType {
		t.Fatalf("text err = %v", err)
	}

	cfg := testConfig(t, t.TempDir())
	cfg.Forum.MaxImagePixels = 1
	if _, err := uploads.Save(pngBytes(t, 4, 4), forummodels.UploadKindPost); err != uploads.ErrInvalidImage {
		t.Fatalf("pixel err = %v", err)
	}

	cfg = testConfig(t, t.TempDir())
	cfg.Forum.MaxUploadBytes = 10
	if _, err := uploads.Save(pngBytes(t, 4, 4), forummodels.UploadKindPost); err != uploads.ErrTooLarge {
		t.Fatalf("size err = %v", err)
	}
}

func TestPathValidation(t *testing.T) {
	testConfig(t, t.TempDir())

	for _, key := range []string{
		"../../etc/passwd",
		"2026/11/nothex.png",
		"2026/11/0123456789abcdef0123456789abcdef.exe",
		"2026/11/0123456789abcdef0123456789abcdef.png/../../x",
	} {
		if _, err := uploads.Path(key); err == nil {
			t.Fatalf("key %q should be rejected", key)
		}
	}

	if _, err := uploads.Path("2026/11/0123456789abcdef0123456789abcdef.png"); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}

	if _, ok := uploads.ParseKind("other"); ok {
		t.Fatal("invalid kind accepted")
	}
}
