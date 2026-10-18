#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

if [ ! -f /etc/apt/sources.list.d/adoptium.list ] || [ ! -s /etc/apt/keyrings/adoptium.asc ]; then
  echo "配置 Adoptium Java 源"

  # https://mirrors.tuna.tsinghua.edu.cn/help/Adoptium/

  apt-get update && apt-get install -y wget apt-transport-https

  mkdir -p /etc/apt/keyrings
  wget -O /etc/apt/keyrings/adoptium.asc https://packages.adoptium.net/artifactory/api/gpg/key/public

  # debian 12
  echo "deb [signed-by=/etc/apt/keyrings/adoptium.asc] https://mirrors.tuna.tsinghua.edu.cn/Adoptium/deb bookworm main" > /etc/apt/sources.list.d/adoptium.list
else
  echo "Adoptium Java 源已配置，跳过写入"
fi

# 无论是否跳过写入都刷新索引：上次部署可能正好在 apt-get update 时失败，
# 此时源文件已经存在，但索引尚未生成，必须在这里补齐。
apt-get update
