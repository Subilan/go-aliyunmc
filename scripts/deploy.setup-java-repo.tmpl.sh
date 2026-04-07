#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

echo "配置 Adoptium Java 源"

# https://mirrors.tuna.tsinghua.edu.cn/help/Adoptium/

apt-get update && apt-get install -y wget apt-transport-https

wget -O - https://packages.adoptium.net/artifactory/api/gpg/key/public | tee /etc/apt/keyrings/adoptium.asc

# debian 12
echo "deb [signed-by=/etc/apt/keyrings/adoptium.asc] https://mirrors.tuna.tsinghua.edu.cn/Adoptium/deb bookworm main" > /etc/apt/sources.list.d/adoptium.list

apt-get update