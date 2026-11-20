// Package bot 提供外部 QQ 机器人接入的认证基础设施。
package bot

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/botkey"
	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"

	"github.com/gin-gonic/gin"
)

// Prefix 是 bot API 的模块前缀。
const Prefix = "/bot-v1"

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// botRuntime 是机器人模块的进程内运行状态。
type botRuntime struct {
	cfg       config.Bot
	keyID     string
	publicKey ed25519.PublicKey
	nonces    *nonceCache
	limiter   *rateLimiter
	now       func() time.Time
}

var globalRuntime *botRuntime

// ValidateConfig 校验机器人配置；未启用时直接通过。
func ValidateConfig() error {
	cfg := config.Get().Bot
	if !cfg.Enabled {
		return nil
	}

	if !keyIDPattern.MatchString(cfg.KeyID) {
		return fmt.Errorf("[bot].key_id %q 不合法，只允许 A-Z a-z 0-9 _ . -，长度 1-64", cfg.KeyID)
	}
	if cfg.PublicKeyPath == "" {
		return fmt.Errorf("[bot].public_key_path 不能为空")
	}
	if _, err := loadPublicKey(cfg.PublicKeyPath); err != nil {
		return err
	}

	return nil
}

func loadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取机器人公钥失败(%s): %w", path, err)
	}

	pub, err := botkey.ParsePublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("解析机器人公钥失败(%s): %w", path, err)
	}

	return pub, nil
}

// newRuntime 根据配置构建运行状态；未启用时返回 nil。
func newRuntime(cfg config.Bot) (*botRuntime, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if !keyIDPattern.MatchString(cfg.KeyID) {
		return nil, fmt.Errorf("[bot].key_id %q 不合法", cfg.KeyID)
	}

	pub, err := loadPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return nil, err
	}

	return &botRuntime{
		cfg:       cfg,
		keyID:     cfg.KeyID,
		publicKey: pub,
		nonces:    newNonceCache(2*time.Duration(cfg.TimestampToleranceSec)*time.Second, cfg.NonceCapacity),
		limiter:   newRateLimiter(cfg.RateLimitPerMinute),
		now:       time.Now,
	}, nil
}

// MustInitialize 初始化机器人模块；未启用时不做任何事。
func MustInitialize() {
	cfg := config.Get().Bot
	if !cfg.Enabled {
		return
	}

	rt, err := newRuntime(cfg)
	if err != nil {
		log_util.Fatal("初始化机器人模块失败: %v", err)
	}

	if !config.Get().Base.TLS.Enabled && !config.Get().Base.SecureCookie {
		log_util.Warn("机器人接口已启用，但未检测到本地 TLS 或 secure_cookie=true；请确保反向代理提供 HTTPS")
	}

	globalRuntime = rt
}

// Bind 注册机器人路由；未启用时不注册任何路由。
func Bind(engine *gin.Engine) {
	if !config.Get().Bot.Enabled || globalRuntime == nil {
		return
	}

	rt := globalRuntime
	router := engine.Group(Prefix)
	router.Use(Verify(rt))
	router.Use(RateLimit(rt))
	bindRoutes(router)
}
