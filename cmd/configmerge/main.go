// Command configmerge 把分片时代的旧配置文件合并成新的 configs/config.toml。
//
// 这是一次性工具，迁移完成后即删除。它只写 config.toml，不碰 store.toml
// （数据库配置在新布局里仍是独立文件，形状未变）。
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"

	"github.com/Subilan/go-aliyunmc/config"
)

// commentedLeaf 匹配被 commented 选项注释掉的叶子键值行，例如 "# region_id = 'cn-hangzhou'"。
// 合并方向（真实值 → 文件）与示例方向相反：这些字段的值必须还原成生效的行。
// 注释行的文字都是中文说明，不会命中（config 包的守护测试保证 commented 只用在叶子上）。
var commentedLeaf = regexp.MustCompile(`(?m)^# ([a-z_]+) = `)

func main() {
	dir := "configs"
	if d := os.Getenv("GO_ALIYUNMC_CONFIG_DIR"); d != "" {
		dir = d
	}

	var cfg config.Config
	var dropped []string

	for _, m := range []struct {
		name    string
		section any
	}{
		{"main", &cfg.Base},
		{"aliyun", &cfg.Aliyun},
		{"server", &cfg.Server},
		{"perms", &cfg.Perms},
		{"remote-util", &cfg.SSH},
		{"user-routes", &cfg.UserRoutes},
		{"task-deploy", &cfg.Tasks.Deploy},
		{"task-backup", &cfg.Tasks.Backup},
		{"task-archive", &cfg.Tasks.Archive},
		{"monitor-server", &cfg.Monitors.Server},
		{"monitor-instance", &cfg.Monitors.Instance},
		{"monitor-file-sync", &cfg.Monitors.FileSync},
		{"monitor-auto-archive-idle", &cfg.Monitors.AutoArchiveIdle},
		{"monitor-backup", &cfg.Monitors.Backup},
		{"monitor-spot-interruption", &cfg.Monitors.SpotInterruption},
		{"monitor-best-ecs-candidate", &cfg.Monitors.BestEcsCandidate},
		{"sampler-player-count", &cfg.Samplers.PlayerCount},
		{"sampler-balance", &cfg.Samplers.Balance},
	} {
		merge(filepath.Join(dir, m.name+".toml"), m.name, m.section, &dropped)
	}

	out, err := toml.Marshal(cfg)
	if err != nil {
		fmt.Printf("序列化失败: %v\n", err)
		os.Exit(1)
	}

	path := filepath.Join(dir, "config.toml")
	text := commentedLeaf.ReplaceAllString(string(out), "$1 = ")
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		fmt.Printf("写入 %s 失败: %v\n", path, err)
		os.Exit(1)
	}

	fmt.Printf("已写入 %s\n", path)

	if len(dropped) > 0 {
		fmt.Println("\n以下旧键在新结构里不存在，已被丢弃：")
		for _, d := range dropped {
			fmt.Println("  " + d)
		}
	}

	verifyStore(filepath.Join(dir, "store.toml"), &config.Store{})
	verifyAll(dir)
}

// verifyAll 复刻 MustLoad 的加载顺序（先 store 后 config，装进同一份 Config），
// 再跑一遍默认值与校验：单独校验 config.toml 会因为 Store 为空而误报。
func verifyAll(dir string) {
	var cfg config.Config

	for _, m := range []struct {
		name    string
		section any
	}{{"store", &cfg.Store}, {"config", &cfg}} {
		path := filepath.Join(dir, m.name+".toml")
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("\n警告：读不到 %s：%v\n", path, err)
			return
		}

		decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
		if err := decoder.Decode(m.section); err != nil {
			fmt.Printf("\n警告：%s 严格解码失败，需要手工清理：%s\n", path, decodeDetail(err))
			return
		}
	}

	if err := defaults.Set(&cfg); err != nil {
		fmt.Printf("\n警告：填充默认值失败：%v\n", err)
		return
	}

	if err := validator.New().Struct(&cfg); err != nil {
		fmt.Printf("\n警告：合并结果不满足校验（缺少必填项，见下）：%v\n", err)
		return
	}

	fmt.Println("合并结果（store.toml + config.toml）严格解码与校验均通过。")
}

// merge 把一个旧文件读进它对应的分节。未知键按"报告并丢弃"处理，不中止迁移。
func merge(path, name string, section any, dropped *[]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("跳过 %s：读不到该文件\n", path)
		return
	}

	// 宽松解码：旧文件里的历史死键不该阻断迁移，它们只是被丢弃。
	if err := toml.Unmarshal(data, section); err != nil {
		fmt.Printf("解析 %s 失败: %v\n", path, err)
		os.Exit(1)
	}

	// 同一份数据再严格解码一次，只为把"新结构里没有的键"列出来。
	probe := reflect.New(reflect.TypeOf(section).Elem()).Interface()
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(probe); err != nil {
		*dropped = append(*dropped, fmt.Sprintf("%s.toml: %s", name, decodeDetail(err)))
	}
}

// verifyStore 用新结构回读一个文件，确认它在严格模式与合法性校验下都能通过。
func verifyStore(path string, target any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("\n警告：读不到 %s：%v\n", path, err)
		return
	}

	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		fmt.Printf("\n警告：%s 严格解码失败，需要手工清理：%s\n", path, decodeDetail(err))
		return
	}

	if err := defaults.Set(target); err != nil {
		fmt.Printf("\n警告：%s 填充默认值失败：%v\n", path, err)
		return
	}

	if err := validator.New().Struct(target); err != nil {
		fmt.Printf("\n警告：%s 不满足校验（可能是缺少必填项）：%v\n", path, err)
		return
	}

	fmt.Printf("%s 严格解码与校验均通过。\n", path)
}

func decodeDetail(err error) string {
	var missing *toml.StrictMissingError
	if errors.As(err, &missing) {
		return strings.Join(strings.Fields(missing.String()), " ")
	}

	return strings.Join(strings.Fields(err.Error()), " ")
}
