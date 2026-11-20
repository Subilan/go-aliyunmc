package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Subilan/go-aliyunmc/internal/common/botkey"
	"github.com/Subilan/go-aliyunmc/internal/common/paths"
)

// runGenBotKey 生成机器人使用的 Ed25519 密钥对。
func runGenBotKey() {
	fs := flag.NewFlagSet("gen_bot_key", flag.ExitOnError)
	outDir := fs.String("out-dir", filepath.Join(paths.RuntimeDir(), "bot"), "密钥输出目录")
	force := fs.Bool("force", false, "覆盖已存在的密钥文件")
	_ = fs.Parse(os.Args[2:])

	privatePath := filepath.Join(*outDir, "bot_private.pem")
	publicPath := filepath.Join(*outDir, "bot_public.pem")

	if !*force {
		for _, path := range []string{privatePath, publicPath} {
			if _, err := os.Stat(path); err == nil {
				fmt.Fprintf(os.Stderr, "错误: %s 已存在，使用 -force 覆盖\n", path)
				os.Exit(1)
			} else if !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "错误: 检查 %s 失败: %v\n", path, err)
				os.Exit(1)
			}
		}
	}

	privatePEM, publicPEM, err := botkey.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 生成密钥失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 创建目录 %s 失败: %v\n", *outDir, err)
		os.Exit(1)
	}
	if err := os.WriteFile(privatePath, privatePEM, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 写入私钥失败: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(publicPath, publicPEM, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 写入公钥失败: %v\n", err)
		os.Exit(1)
	}

	pub, err := botkey.ParsePublicKey(publicPEM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 校验生成的公钥失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("私钥: %s\n", privatePath)
	fmt.Printf("公钥: %s\n", publicPath)
	fmt.Printf("公钥指纹: sha256:%s\n", botkey.Fingerprint(pub))
	fmt.Println("把私钥交给机器人；服务端 [bot].public_key_path 指向公钥文件。")
}
