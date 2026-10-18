#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"
ARCHIVE_DIR="${USER_HOME}/server/archive"
TMP_DIR="${USER_HOME}/server/.archive.tmp"

echo "复制归档数据"

# 先下载到临时目录，校验通过后再原子替换目标目录。
# 这样失败时 archive 要么是上一次的完整副本，要么不存在，不会留下半份存档。
rm -rf "${TMP_DIR}"
mkdir -p "${TMP_DIR}"
trap 'rm -rf "${TMP_DIR}"' EXIT

ossutil cp --parallel=7 --jobs=25 -r -f "{{ .ArchiveOSSPath }}/" "${TMP_DIR}" > "${USER_HOME}/restore_archive_data.log" 2>&1

for required in start.sh boot.sh; do
  if [ ! -f "${TMP_DIR}/${required}" ]; then
    echo "归档校验失败：缺少 ${required}"
    exit 1
  fi
done

chmod +x "${TMP_DIR}/boot.sh" "${TMP_DIR}/start.sh"

rm -rf "${ARCHIVE_DIR}"
mv "${TMP_DIR}" "${ARCHIVE_DIR}"
trap - EXIT

chown -R "${USERNAME}:${USERNAME}" "${USER_HOME}"
