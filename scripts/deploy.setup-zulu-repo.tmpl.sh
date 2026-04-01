#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

echo "配置 Zulu Java 源"
apt install -y gnupg ca-certificates curl

rm /usr/share/keyrings/azul.gpg || true

curl -s https://repos.azul.com/azul-repo.key \
  | gpg --batch --dearmor -o /usr/share/keyrings/azul.gpg

echo "deb [signed-by=/usr/share/keyrings/azul.gpg] https://repos.azul.com/zulu/deb stable main" \
  | tee /etc/apt/sources.list.d/zulu.list

chmod 644 /usr/share/keyrings/azul.gpg
apt update -y
