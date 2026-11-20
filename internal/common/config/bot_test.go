package config

import "testing"

func TestBotDefaults(t *testing.T) {
	cfg, err := loadAll(t, validConfig)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	if cfg.Bot.Enabled {
		t.Error("bot 默认不应启用")
	}
	if cfg.Bot.RateLimitPerMinute != 120 {
		t.Errorf("rate_limit_per_minute 默认值不正确: %d", cfg.Bot.RateLimitPerMinute)
	}
	if cfg.Bot.API.Port != 8081 {
		t.Errorf("api.port 默认值不正确: %d", cfg.Bot.API.Port)
	}
}
