#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

echo "复制归档数据"

ossutil cp --parallel=7 --jobs=25 -r -f "{{ .ArchiveOSSPath }}/" "${USER_HOME}/server/archive" > "${USER_HOME}/restore_archive_data.log" 2>&1
chmod +x "${USER_HOME}/server/archive/boot.sh"
chmod +x "${USER_HOME}/server/archive/start.sh"

chown -R "${USERNAME}:${USERNAME}" "${USER_HOME}"
