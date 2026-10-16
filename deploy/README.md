# 发布与远端布局

## 目录

- `systemd/go-aliyunmc.service`：远端 systemd 服务的唯一事实源。
- `package-public.sh`：主仓库 CI 用来打包“公共发布物”（二进制、RBAC、Minecraft 语言文件、脚本模板、systemd unit），不含生产配置。
- `remote-install.sh`：在远端执行（root 或具备白名单 sudo 的 `gomc`）；校验、安装 release、切换 `current`、重启 systemd、健康检查、失败回滚、裁剪旧 release。
- `migrate-remote-layout.sh`：一次性把远端从平铺布局迁移到 `releases/<version> + current` 布局。
- `sudoers/go-aliyunmc` + `install-gomc-sudoers.sh`：允许 `gomc` 通过白名单 sudo 安装 unit 和重启服务，因此 `DEPLOY_USER` 可以配置为 `gomc` 而不是 root。

如果 `DEPLOY_USER=gomc`，需要先执行一次 `deploy/install-gomc-sudoers.sh`。

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

## Release 保留策略

`remote-install.sh` 在健康检查通过后按修改时间裁剪 `releases/`，保留最新的 `KEEP_RELEASES` 个（默认 3，含 `current` 指向的那个）。`current` 无论多旧都不会被删除，因此回滚目标始终可用；清理只在发布成功后执行，发布失败时不裁剪，避免回滚时缺少目标。

可以通过环境变量覆盖：

```bash
KEEP_RELEASES=5 bash remote-install.sh <release-name>
```

同时会清理 `.deploy/` 下滞留超过一天的暂存目录。
