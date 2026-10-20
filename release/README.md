# 发布契约

本目录是主仓库与 production 仓库之间唯一的发布契约。production 只消费 `make bundle` 的输出，不关心主仓库其余目录如何组织。

## 生成发布包

```bash
make bundle OUT=/tmp/bundle
```

输出：

```text
/tmp/bundle/
├── go-aliyunmc                 # linux/amd64 二进制
├── assets/rbac/{model.conf,policy.csv}
├── assets/scripts/
├── go-aliyunmc.service
└── remote-install.sh
```

production 仓库在 bundle 之上叠加 `config/`，生成 `deploy-manifest.json` 与 `SHA256SUMS`，然后上传远端。

## 文件职责

- `bundle.sh`：打包逻辑的唯一实现，主仓库 CI 与 production workflow 都调用它。
- `remote-install.sh`：远端安装脚本。校验、移入 `releases/`、切换 `current`、重启、健康检查、失败回滚、裁剪旧 release。
- `systemd/go-aliyunmc.service`：远端 systemd unit。
- `sudoers/go-aliyunmc`：`gomc` 的白名单 sudo 规则。安装脚本在 production 仓库的 `bootstrap/` 下。

## 远端布局

```text
/home/gomc/prod/
├── current -> releases/<version>
├── releases/<version>/
│   ├── go-aliyunmc
│   ├── config/{config.toml,store.toml}
│   ├── assets/{rbac,scripts}
│   ├── go-aliyunmc.service
│   ├── remote-install.sh
│   ├── deploy-manifest.json
│   └── SHA256SUMS
├── config -> current/config
├── assets -> current/assets
├── go-aliyunmc -> current/go-aliyunmc
└── state/{logs,cache,ecs_candidates.json}
```

`config` 与 `assets` 跟随 `current`，回滚时一并回滚。`state/` 是运行时数据，发布不触碰。

从旧布局（`configs/`、`scripts/`、`rbac_*`、`minecraft_*`、`logs/`、`remote_data_cache/`）迁移的命令只执行一次，见 production 仓库的 `DEPLOYMENT.md`。安装脚本本身不含任何兼容逻辑。
