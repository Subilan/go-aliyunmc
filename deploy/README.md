# 发布与远端布局

## 目录

- `systemd/go-aliyunmc.service`：远端 systemd 服务的唯一事实源。
- `package-public.sh`：主仓库 CI 用来打包“公共发布物”（二进制、RBAC、Minecraft 语言文件、脚本模板、systemd unit），不含生产配置。
- `remote-install.sh`：在远端以 root 执行；校验、安装 release、切换 `current`、重启 systemd、健康检查、失败回滚。
- `migrate-remote-layout.sh`：一次性把远端从平铺布局迁移到 `releases/<version> + current` 布局。

## 远端目标布局

```text
/home/gomc/prod/
├── current -> releases/<version>
├── releases/<version>/
│   ├── go-aliyunmc
│   ├── configs/
│   ├── rbac_model.conf
│   ├── rbac_policy.csv
│   ├── minecraft_*.json
│   ├── scripts/
│   ├── go-aliyunmc.service
│   ├── remote-install.sh
│   ├── deploy-manifest.json
│   └── SHA256SUMS
├── logs/                 # 运行时状态，发布不触碰
├── remote_data_cache/
└── ecs_candidates.json
```
