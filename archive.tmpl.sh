#!/usr/bin/env bash

set -euo pipefail

# ===== 配置项 =====
LOCAL_ARCHIVE_DIR="/home/mc/server/archive"
OSS_BUCKET="{{ .ArchiveOSSPath }}"
OSS_ARCHIVE_DIR="${OSS_BUCKET}/archive"

# ===== 基本校验 =====
if [[ ! -d "${LOCAL_ARCHIVE_DIR}" ]]; then
    echo "ERROR: 本地目录不存在: ${LOCAL_ARCHIVE_DIR}" >&2
    exit 1
fi

# ===== 执行归档 =====
echo "开始归档：${LOCAL_ARCHIVE_DIR} -> ${OSS_ARCHIVE_DIR}"

ossutil cp -r \
    "${LOCAL_ARCHIVE_DIR}" \
    "${OSS_ARCHIVE_DIR}"

echo "归档完成"