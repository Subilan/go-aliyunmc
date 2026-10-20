// Package mcassets 提供编译进二进制的 Minecraft 语言文件。
package mcassets

import _ "embed"

//go:embed en_us.json
var EnUS []byte

//go:embed zh_cn.json
var ZhCN []byte
