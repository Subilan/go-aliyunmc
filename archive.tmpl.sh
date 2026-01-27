#!/usr/bin/env bash

set -euo pipefail

# ===== 配置项 =====
LOCAL_ARCHIVE_DIR="/home/mc/server/archive"
OSS_BUCKET="{{ .OSSRoot }}"

ARCHIVE="archive"
ARCHIVE_NEW="archive-new"
ARCHIVE_OLD="archive-old"

has_objects() {
    ossutil ls "$1" | awk '/Object Number is:/ {print $4}' | grep -qv '^0$'
}

# ===== 基本校验 =====
if [[ ! -d "${LOCAL_ARCHIVE_DIR}" ]]; then
    echo "ERROR: 本地目录不存在: ${LOCAL_ARCHIVE_DIR}" >&2
    exit 1
fi

# ===== Step 1: 上传到 archive-new =====
echo "上传新归档 -> ${ARCHIVE_NEW}"

ossutil cp -r -f \
    "${LOCAL_ARCHIVE_DIR}" \
    "${OSS_BUCKET}/${ARCHIVE_NEW}/"

# ===== Step 2: 删除已有 archive-old =====
if has_objects "${OSS_BUCKET}/${ARCHIVE_OLD}/"; then
    echo "删除旧的 ${ARCHIVE_OLD}"
    ossutil rm -rf "${OSS_BUCKET}/${ARCHIVE_OLD}/"
fi

# ===== Step 3: archive -> archive-old（复制）=====
if has_objects "${OSS_BUCKET}/${ARCHIVE}/"; then
    echo "复制 ${ARCHIVE} -> ${ARCHIVE_OLD}"
    ossutil cp -r -f \
        "${OSS_BUCKET}/${ARCHIVE}/" \
        "${OSS_BUCKET}/${ARCHIVE_OLD}/"

    echo "删除原 ${ARCHIVE}"
    ossutil rm -rf "${OSS_BUCKET}/${ARCHIVE}/"
else
    echo "未发现 ${ARCHIVE}，跳过 archive-old 生成"
fi

# ===== Step 4: archive-new -> archive =====
echo "复制 ${ARCHIVE_NEW} -> ${ARCHIVE}"

ossutil cp -r -f \
    "${OSS_BUCKET}/${ARCHIVE_NEW}/" \
    "${OSS_BUCKET}/${ARCHIVE}/"

echo "删除 ${ARCHIVE_NEW}"
ossutil rm -rf "${OSS_BUCKET}/${ARCHIVE_NEW}/"

echo "归档轮转完成"