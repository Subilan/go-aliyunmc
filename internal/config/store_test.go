package config

import (
	"testing"

	"github.com/creasty/defaults"
)

// 这些用例先过一遍默认值填充，与运行时的加载顺序一致。
func TestStoreDSNDefaultValues(t *testing.T) {
	cfg := Store{}
	if err := defaults.Set(&cfg); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}

	want := "root:@tcp(localhost:3306)/aliyunmc?charset=utf8mb4&parseTime=True&loc=Local"
	if got := cfg.DSN(); got != want {
		t.Errorf("DSN 生成错误，期望: %s, 实际: %s", want, got)
	}
}

func TestStoreDSNCustomValues(t *testing.T) {
	cfg := Store{
		Host:     "db.example.com",
		Port:     3307,
		User:     "user",
		Password: "pass",
		DBName:   "mydb",
		Charset:  "utf8",
	}
	if err := defaults.Set(&cfg); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}

	want := "user:pass@tcp(db.example.com:3307)/mydb?charset=utf8&parseTime=True&loc=Local"
	if got := cfg.DSN(); got != want {
		t.Errorf("DSN 生成错误，期望: %s, 实际: %s", want, got)
	}
}
