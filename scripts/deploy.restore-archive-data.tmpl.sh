#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

echo "复制归档数据"

ossutil cp -r "{{ .ArchiveOSSPath }}" "${USER_HOME}/server"
chmod +x "${USER_HOME}/server/archive/boot.sh"
chmod +x "${USER_HOME}/server/archive/start.sh"

chown -R "${USERNAME}:${USERNAME}" "${USER_HOME}"

echo "===== 部署成功完成 ====="
