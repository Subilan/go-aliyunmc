#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

echo "安装系统软件"
apt install -y temurin-{{ .JavaVersion }}-jre {{ range .Packages }}{{ . }} {{ end }}
