package config

import (
	"strings"
	"testing"
)

func TestBotDefaults(t *testing.T) {
	cfg, err := loadAll(t, validConfig)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	if cfg.Bot.Enabled {
		t.Error("bot 默认不应启用")
	}
	if cfg.Bot.KeyID != "default" {
		t.Errorf("key_id 默认值不正确: %q", cfg.Bot.KeyID)
	}
	if cfg.Bot.TimestampToleranceSec != 300 {
		t.Errorf("timestamp_tolerance_sec 默认值不正确: %d", cfg.Bot.TimestampToleranceSec)
	}
	if cfg.Bot.NonceCapacity != 10000 {
		t.Errorf("nonce_capacity 默认值不正确: %d", cfg.Bot.NonceCapacity)
	}
	if cfg.Bot.MaxBodyBytes != 1048576 {
		t.Errorf("max_body_bytes 默认值不正确: %d", cfg.Bot.MaxBodyBytes)
	}
	if cfg.Bot.RateLimitPerMinute != 120 {
		t.Errorf("rate_limit_per_minute 默认值不正确: %d", cfg.Bot.RateLimitPerMinute)
	}
}

func TestBotRequiresPublicKeyWhenEnabled(t *testing.T) {
	_, err := loadAll(t, validConfig+"\n[bot]\nenabled = true\n")
	if err == nil {
		t.Fatal("启用 bot 但缺少 public_key_path 时应当报错")
	}
	if !strings.Contains(err.Error(), "PublicKeyPath") {
		t.Errorf("报错未指出 PublicKeyPath: %v", err)
	}
}

func TestBotRejectsUnknownScope(t *testing.T) {
	_, err := loadAll(t, validConfig+"\n[bot]\nscopes = [\"root\"]\n")
	if err == nil {
		t.Fatal("非法 scope 应当报错")
	}
}

func TestBotAcceptsKnownScopes(t *testing.T) {
	cfg, err := loadAll(t, validConfig+"\n[bot]\nscopes = [\"read\", \"account\", \"admin\"]\n")
	if err != nil {
		t.Fatalf("合法 scope 不应报错: %v", err)
	}
	if len(cfg.Bot.Scopes) != 3 {
		t.Fatalf("scope 数量不正确: %v", cfg.Bot.Scopes)
	}
}
