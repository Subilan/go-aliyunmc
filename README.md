# go-aliyunmc

[![Go Reference](https://pkg.go.dev/badge/github.com/Subilan/go-aliyunmc.svg)](https://pkg.go.dev/github.com/Subilan/go-aliyunmc)
<!--[![Go](https://github.com/Subilan/go-aliyunmc/actions/workflows/go.yml/badge.svg)](https://github.com/Subilan/go-aliyunmc/actions/workflows/go.yml)-->
go-aliyunmc 是一个基于阿里云 ECS 的 Minecraft 服务器低成本解决方案，提供相比面板更高的自主性和透明性、相比独立服务器更低廉的成本投入。

go-aliyunmc 是 [Seatide](https://seatide.net) 服务器的运行基石，是该服务器自 2021 年起运行模式的第三次迭代（第一、二版分别于 2021 年使用 Python 语言、2024 年使用 Go 语言构建）。

特点：
- **轻量**，使用 Go 语言编写，编译后文件无额外依赖；文件运行时占用小，可在 1 GiB RAM + 2 vCPU 环境下稳定高效运行。
- **高可配置性**，提供较多可配置项，部署脚本模板化（见各个 `*.tmpl.sh`），支持代入配置文件项目。
- **自动化**，系统以常驻的不同功能监控线程为核心，对外提供信息 CRUD 服务的同时全面监控受控实例以及实例上运行的服务器状态。
- **系统耦合低**，相比于前两版，go-aliyunmc 提供了最低的耦合程度，无需额外安装 Java 编写的插件（Web Chat 双向通信除外），也无需调用阿里云的 InvokeCommand。远程控制主要通过 Web 请求和 SSH 实现，支持添加实例指令（shell）和 Minecraft 指令（RCON）以远程运行。

## Help Wanted

如果你对基于 go-aliyunmc 的 Minecraft 服务器感兴趣，欢迎加入 Seatide 服务器，该服务器于 2026 年 1 月重新开启。

如果你对 go-aliyunmc 的代码感兴趣，欢迎来协助开发或捉虫。代码中涉及的并发代码可能有许多我未发现的严重问题，代码质量、注释详细程度有待提高，文档也有待完善。

## 前端

当前项目中的代码承担 go-aliyunmc 的主要功能，作为一个完整系统的后端存在。支持本后端的前端包括：
- [go-aliyunmc-client-web-next](https://github.com/Subilan/go-aliyunmc-client-web-next)
- ~~[go-aliyunmc-client-web](https://github.com/Subilan/go-aliyunmc-client-web)~~

## 协议 & 鸣谢

MIT

- 特别感谢：运行模式的灵感始于 [SomeBottle](https://github.com/SomeBottle) 的个人服务器。
- 感谢在 2021-2024 年间支持过 Seatide 的 12 名玩家。
