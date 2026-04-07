#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

echo "复制归档数据"

ossutil cp --parallel=5 --jobs=20 -r "{{ .ArchiveOSSPath }}/" "${USER_HOME}/server/archive"
chmod +x "${USER_HOME}/server/archive/boot.sh"
chmod +x "${USER_HOME}/server/archive/start.sh"

chown -R "${USERNAME}:${USERNAME}" "${USER_HOME}"
