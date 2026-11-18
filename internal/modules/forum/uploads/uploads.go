// Package uploads 负责论坛图片的本地磁盘存储与图片校验。
package uploads

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/paths"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	_ "golang.org/x/image/webp"
)

const (
	maxDimension = 8192
	// maxConcurrentDecodes 限制同时解码的图片数量，避免大图并发占用过多内存。
	maxConcurrentDecodes = 4
	// URLPrefix 是静态文件路由前缀。
	URLPrefix = "/forum-v1/files/"
)

var decodeSlots = make(chan struct{}, maxConcurrentDecodes)

// acquireDecodeSlot 获取一个解码槽位，返回释放函数。
func acquireDecodeSlot() func() {
	decodeSlots <- struct{}{}
	return func() { <-decodeSlots }
}

var keyRe = regexp.MustCompile(`^\d{4}/\d{2}/[a-f0-9]{32}\.(?:jpg|png|webp)$`)

var (
	// ErrUnsupportedType 表示图片格式不在允许列表内。
	ErrUnsupportedType = errors.New("仅支持 JPEG、PNG、WebP 图片")
	// ErrTooLarge 表示图片字节数超过限制。
	ErrTooLarge = errors.New("图片体积超过限制")
	// ErrInvalidImage 表示图片尺寸或内容不合格。
	ErrInvalidImage = errors.New("图片尺寸或内容不合格")
)

// SavedFile 是一次成功写入磁盘的上传结果。
type SavedFile struct {
	Key       string
	Mime      string
	SizeBytes int64
	Width     int
	Height    int
	SHA256    string
}

// Root 返回上传根目录。配置为空时使用 $STATE_DIR/state/forum/uploads。
func Root() string {
	dir := strings.TrimSpace(config.Get().Forum.UploadDir)
	if dir == "" {
		dir = filepath.Join(paths.RuntimeDir(), "forum", "uploads")
	}
	return dir
}

// MustInitialize 创建并校验上传目录；失败直接终止进程。
func MustInitialize() {
	root := Root()
	if err := os.MkdirAll(root, 0750); err != nil {
		log_util.Fatal("创建论坛上传目录失败: %v", err)
	}

	testFile, err := os.CreateTemp(root, ".write-test-*")
	if err != nil {
		log_util.Fatal("论坛上传目录不可写: %v", err)
	}
	name := testFile.Name()
	_ = testFile.Close()
	_ = os.Remove(name)
}

// URL 返回上传文件的公开访问 URL。
func URL(key string) string {
	return URLPrefix + key
}

// Path 校验 key 并返回磁盘路径，防止路径穿越。
func Path(key string) (string, error) {
	key = strings.TrimPrefix(key, "/")
	if !keyRe.MatchString(key) {
		return "", ErrInvalidImage
	}
	return filepath.Join(Root(), filepath.FromSlash(key)), nil
}

// Delete 删除一个上传文件；文件不存在视为成功。
func Delete(key string) error {
	path, err := Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// MaxBytesForKind 返回指定用途的最大字节数。
func MaxBytesForKind(kind forummodels.UploadKind) int64 {
	cfg := config.Get().Forum
	switch kind {
	case forummodels.UploadKindAvatar:
		return cfg.MaxAvatarBytes
	case forummodels.UploadKindBackground:
		return cfg.MaxBackgroundBytes
	default:
		return cfg.MaxUploadBytes
	}
}

// MaxRequestBytes 返回所有上传用途中最大的字节数，用于 HTTP 请求体上限。
func MaxRequestBytes() int64 {
	cfg := config.Get().Forum
	limit := cfg.MaxUploadBytes
	if cfg.MaxAvatarBytes > limit {
		limit = cfg.MaxAvatarBytes
	}
	if cfg.MaxBackgroundBytes > limit {
		limit = cfg.MaxBackgroundBytes
	}
	return limit
}

// ParseKind 校验上传用途。
func ParseKind(value string) (forummodels.UploadKind, bool) {
	switch forummodels.UploadKind(value) {
	case forummodels.UploadKindPost, forummodels.UploadKindAvatar, forummodels.UploadKindBackground:
		return forummodels.UploadKind(value), true
	default:
		return "", false
	}
}

// Save 校验、重编码并写入图片，返回存储结果。
func Save(data []byte, kind forummodels.UploadKind) (*SavedFile, error) {
	if max := MaxBytesForKind(kind); max > 0 && int64(len(data)) > max {
		return nil, ErrTooLarge
	}

	mime := http.DetectContentType(data)
	var ext string
	switch mime {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/webp":
		ext = "webp"
	default:
		return nil, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrInvalidImage
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension {
		return nil, ErrInvalidImage
	}
	if limit := config.Get().Forum.MaxImagePixels; limit > 0 && cfg.Width*cfg.Height > limit {
		return nil, ErrInvalidImage
	}

	out := data
	switch mime {
	case "image/jpeg":
		encoded, err := reencodeJPEG(data)
		if err != nil {
			return nil, err
		}
		out = encoded
	case "image/png":
		encoded, err := reencodePNG(data)
		if err != nil {
			return nil, err
		}
		out = encoded
	}

	sum := sha256.Sum256(out)
	key, err := newKey(ext)
	if err != nil {
		return nil, err
	}

	if err := writeAtomic(key, out); err != nil {
		return nil, err
	}

	return &SavedFile{
		Key:       key,
		Mime:      mime,
		SizeBytes: int64(len(out)),
		Width:     cfg.Width,
		Height:    cfg.Height,
		SHA256:    hex.EncodeToString(sum[:]),
	}, nil
}

// reencodeJPEG 解码并重新编码 JPEG，同时剥离 EXIF 等元数据。
func reencodeJPEG(data []byte) ([]byte, error) {
	release := acquireDecodeSlot()
	defer release()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImage
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("重新编码 JPEG 失败: %w", err)
	}
	return buf.Bytes(), nil
}

// reencodePNG 解码并重新编码 PNG，保留透明通道并剥离元数据。
func reencodePNG(data []byte) ([]byte, error) {
	release := acquireDecodeSlot()
	defer release()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImage
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("重新编码 PNG 失败: %w", err)
	}
	return buf.Bytes(), nil
}

func newKey(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成上传文件名失败: %w", err)
	}
	return time.Now().UTC().Format("2006/01") + "/" + hex.EncodeToString(buf) + "." + ext, nil
}

func writeAtomic(key string, data []byte) error {
	path, err := Path(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("创建上传子目录失败: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入上传文件失败: %w", err)
	}
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("保存上传文件失败: %w", err)
	}

	return nil
}
