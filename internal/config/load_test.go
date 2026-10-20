package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validStore / validConfig 是一份满足全部必填项的最小配置，用例在其中做局部改动。
const validStore = `
dbname = "aliyunmc"
`

const validConfig = `
[server]
rcon_password = "test-rcon-password"

[aliyun]
region_id = "cn-hangzhou"
access_key_id = "test-akid"
access_key_secret = "test-aksecret"

[aliyun.ecs]
image_id = "test-image"
security_group_id = "test-security-group"
root_password = "test-root-password"
prod_password = "test-prod-password"

[aliyun.ecs.system_disk]
size = 40

[aliyun.ecs.data_disk]
size = 100

[ssh]
private_key_path = "/tmp/test-key"

[user_routes.chat_token]
secret = "test-jwt-secret"

[tasks.deploy.vars]
ssh_access_public_key = "ssh-ed25519 AAAA test"
archive_oss_path = "oss://test-bucket/archive"

[tasks.backup.vars]
backup_oss_path = "oss://test-bucket/backup"
target_dirs = ["world"]

[tasks.archive.vars]
oss_root = "oss://test-bucket"

[monitors.best_ecs_candidate]
mem_choices = [8, 16]
cpu_core_count_choices = [4, 8]
`

// writeConfigs 把若干配置文件写进同一个临时目录，并把 STATE_DIR 指向它。
func writeConfigs(t *testing.T, files map[string]string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0755); err != nil {
		t.Fatalf("创建 config 目录失败: %v", err)
	}
	t.Setenv("GO_ALIYUNMC_STATE_DIR", dir)

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, "config", name+".toml"), []byte(content), 0600); err != nil {
			t.Fatalf("写入 %s.toml 失败: %v", name, err)
		}
	}
}

// loadAll 复刻 MustLoad 的加载顺序（先 store 后 config），返回结果或错误。
func loadAll(t *testing.T, configTOML string) (*Config, error) {
	t.Helper()

	writeConfigs(t, map[string]string{"store": validStore, "config": configTOML})

	var cfg Config
	if err := load("store", &cfg.Store); err != nil {
		return nil, err
	}
	if err := load("config", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := loadAll(t, validConfig)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"base.expose", cfg.Base.Expose, 45678},
		{"store.port", cfg.Store.Port, 3306},
		{"server.port", int(cfg.Server.Port), 25565},
		{"monitors.server.poll_interval_sec", cfg.Monitors.Server.PollIntervalSec, 5},
		{"monitors.auto_archive_idle.idle_countdown_sec", cfg.Monitors.AutoArchiveIdle.IdleCountdownSec, 1200},
		{"monitors.best_ecs_candidate.poll_interval_sec", cfg.Monitors.BestEcsCandidate.PollIntervalSec, 600},
		{"tasks.deploy.vars.java_version", cfg.Tasks.Deploy.Vars.JavaVersion, 21},
		{"tasks.backup.timeout_sec", cfg.Tasks.Backup.TimeoutSec, 1800},
		{"tasks.archive.timeout_sec", cfg.Tasks.Archive.TimeoutSec, 600},
		{"samplers.player_count.max_data_points", cfg.Samplers.PlayerCount.MaxDataPoints, 100},
		{"samplers.balance.sample_interval_sec", cfg.Samplers.Balance.SampleIntervalSec, 300},
	} {
		if tc.got != tc.want {
			t.Errorf("%s 的默认值未生效: 得到 %d，期望 %d", tc.name, tc.got, tc.want)
		}
	}

	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{"store.charset", cfg.Store.Charset, "utf8mb4"},
		{"tasks.backup.vars.base_dir", cfg.Tasks.Backup.Vars.BaseDir, "/home/mc/server/archive"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s 的默认值未生效: 得到 %q，期望 %q", tc.name, tc.got, tc.want)
		}
	}
}

// 显式写下的值不能被默认值覆盖。
func TestLoadPreservesExplicitValues(t *testing.T) {
	// port 必须写在已有的 [server] 节内，TOML 不允许重复定义。
	explicit := strings.Replace(validConfig, "[server]\n", "[server]\nport = 1234\n", 1)

	cfg, err := loadAll(t, explicit+`
[base]
expose = 12345

[monitors.auto_archive_idle]
enabled = false
idle_countdown_sec = 7
`)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	if cfg.Base.Expose != 12345 {
		t.Errorf("显式的 expose 被覆盖: %d", cfg.Base.Expose)
	}
	if cfg.Server.Port != 1234 {
		t.Errorf("显式的 port 被覆盖: %d", cfg.Server.Port)
	}
	if cfg.Monitors.AutoArchiveIdle.IdleCountdownSec != 7 {
		t.Errorf("显式的 idle_countdown_sec 被覆盖: %d", cfg.Monitors.AutoArchiveIdle.IdleCountdownSec)
	}
	if cfg.Monitors.AutoArchiveIdle.Enabled {
		t.Error("显式的 enabled = false 被翻转")
	}
}

func TestLoadRejectsUnknownKey(t *testing.T) {
	_, err := loadAll(t, validConfig+`
[monitors.auto_archive_idle]
idle_countdown_secs = 10
`)
	if err == nil {
		t.Fatal("未知键应当报错")
	}
	if !strings.Contains(err.Error(), "idle_countdown_secs") {
		t.Errorf("报错未指出是哪个键: %v", err)
	}
}

func TestLoadRejectsMissingRequired(t *testing.T) {
	_, err := loadAll(t, "")
	if err == nil {
		t.Fatal("缺少必填项应当报错")
	}
	if !strings.Contains(err.Error(), "RegionId") {
		t.Errorf("报错未指出缺了哪个字段: %v", err)
	}
}

// store.toml 只含数据库配置，其余各节缺失也必须能加载。
func TestLoadStoreOnlyNeedsStoreSection(t *testing.T) {
	writeConfigs(t, map[string]string{"store": validStore})

	var cfg Config
	if err := load("store", &cfg.Store); err != nil {
		t.Fatalf("只读 store 节时不应报错: %v", err)
	}
	want := "root:@tcp(localhost:3306)/aliyunmc?charset=utf8mb4&parseTime=True&loc=Local"
	if cfg.Store.DSN() != want {
		t.Errorf("DSN 不正确: %s", cfg.Store.DSN())
	}
}

func TestLoadStoreRejectsInvalidPort(t *testing.T) {
	writeConfigs(t, map[string]string{"store": "port = 70000\n"})

	var cfg Config
	if err := load("store", &cfg.Store); err == nil {
		t.Fatal("超出范围的 port 应当报错")
	}
}

func TestGetPanicsBeforeLoad(t *testing.T) {
	original := current.Load()
	t.Cleanup(func() { current.Store(original) })
	current.Store(nil)

	defer func() {
		if recover() == nil {
			t.Error("尚未加载配置时 Get() 应当 panic")
		}
	}()

	Get()
}
