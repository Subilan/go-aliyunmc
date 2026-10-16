// Command configgen 依据 config 包里的结构体与它们的 tag 生成示例配置。
//
// 示例完全由 tag 驱动：comment tag 变成注释，default tag 变成默认值，
// commented 选项把必须自行填写的字段（凭证、密钥路径、资源 ID 等）注释掉。
// 因此新增字段会自动出现在示例里，不需要在这里补任何东西。
package main

import (
	"fmt"
	"os"

	"github.com/creasty/defaults"
	"github.com/pelletier/go-toml/v2"

	"github.com/Subilan/go-aliyunmc/config"
)

func main() {
	const dir = "example_configs"

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建 %s 失败: %v\n", dir, err)
		os.Exit(1)
	}

	var cfg config.Config
	if err := defaults.Set(&cfg); err != nil {
		fmt.Printf("填充默认值失败: %v\n", err)
		os.Exit(1)
	}

	// store 单独一个文件：run / migrate / create_user 三个入口都需要它，
	// 而另外两个不该被云凭证之类的配置牵着走。Config.Store 标了 toml:"-"，
	// 所以写 config.toml 时它不会出现。
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
