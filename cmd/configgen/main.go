// Command configgen 依据 config 包结构体上的 tag 生成示例配置。
package main

import (
	"fmt"
	"os"

	"github.com/creasty/defaults"
	"github.com/pelletier/go-toml/v2"

	"github.com/Subilan/go-aliyunmc/internal/config"
)

func main() {
	const dir = "examples"

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建 %s 失败: %v\n", dir, err)
		os.Exit(1)
	}

	var cfg config.Config
	if err := defaults.Set(&cfg); err != nil {
		fmt.Printf("填充默认值失败: %v\n", err)
		os.Exit(1)
	}

	// store 单独写成 store.toml；Config.Store 标了 toml:"-"，不会出现在 config.toml 里。
	if err := write(dir+"/store.toml", cfg.Store); err != nil {
		fmt.Printf("写入 store.toml 失败: %v\n", err)
		os.Exit(1)
	}

	if err := write(dir+"/config.toml", cfg); err != nil {
		fmt.Printf("写入 config.toml 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("示例配置已生成到 " + dir)
}

func write(path string, obj any) error {
	out, err := toml.Marshal(obj)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}
