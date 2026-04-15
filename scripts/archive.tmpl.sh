#!/usr/bin/env bash

set -euo pipefail

# ===== 配置项 =====
LOCAL_ARCHIVE_DIR="{{ .RemoteArchiveDir }}"
OSS_BUCKET="{{ .OSSRoot }}"

ARCHIVE="{{ .ArchiveName }}"
ARCHIVE_NEW="{{ .ArchiveName }}-new"
ARCHIVE_OLD="{{ .ArchiveName }}-old"
J=20
P=4

has_objects() {
    ossutil ls "$1" | awk '/Object Number is:/ {print $4}' | grep -qv '^0$'
}

oss_cp() {
  ossutil --jobs=$J --parallel=$P cp -r -f $1 $2
}

# ===== 基本校验 =====
if [[ ! -d "${LOCAL_ARCHIVE_DIR}" ]]; then
    echo "ERROR: 本地目录不存在: ${LOCAL_ARCHIVE_DIR}" >&2
    exit 1
fi

# ===== Step 1: 上传到 archive-new =====
echo "上传新归档 -> ${ARCHIVE_NEW}"

oss_cp \
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
    oss_cp \
        "${OSS_BUCKET}/${ARCHIVE}/" \
        "${OSS_BUCKET}/${ARCHIVE_OLD}/"

    echo "删除原 ${ARCHIVE}"
    ossutil rm -rf "${OSS_BUCKET}/${ARCHIVE}/"
else
    echo "未发现 ${ARCHIVE}，跳过 archive-old 生成"
fi

# ===== Step 4: archive-new -> archive =====
echo "复制 ${ARCHIVE_NEW} -> ${ARCHIVE}"

oss_cp \
    "${OSS_BUCKET}/${ARCHIVE_NEW}/" \
    "${OSS_BUCKET}/${ARCHIVE}/"

echo "删除 ${ARCHIVE_NEW}"
ossutil rm -rf "${OSS_BUCKET}/${ARCHIVE_NEW}/"

echo "归档轮转完成"
