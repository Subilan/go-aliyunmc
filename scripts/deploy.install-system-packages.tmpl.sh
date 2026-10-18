#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

echo "安装系统软件"

# 上一次部署可能在中途被中断，留下未完成配置的包。
# 先恢复 dpkg 状态，保证重试时 apt 能从当前状态继续。
dpkg --configure -a || true
apt-get -f install -y || true

apt install -y temurin-{{ .JavaVersion }}-jre {{ range .Packages }}{{ . }} {{ end }}
