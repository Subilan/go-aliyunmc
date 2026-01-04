#!/usr/bin/env bash

set -euo pipefail

# ===== 配置项 =====
BASE_DIR="/home/mc/server/archive"
BACKUP_ITEMS=(
    "world"
    "world_nether"
    "world_the_end"
)

TMP_DIR="/home/mc/mc_backup"
MAX_BACKUPS=5

OSS_BACKUP_DIR="{{ .BackupOSSPath }}"

# ===== 时间戳 =====
TIMESTAMP="$(date +"%Y%m%d_%H%M%S")"
ZIP_NAME="${TIMESTAMP}.zip"
ZIP_PATH="${TMP_DIR}/${ZIP_NAME}"

# ===== 准备临时目录 =====
mkdir -p "${TMP_DIR}"

# ===== 校验目录 =====
for item in "${BACKUP_ITEMS[@]}"; do
    if [[ ! -d "${BASE_DIR}/${item}" ]]; then
        echo "ERROR: 目录不存在: ${BASE_DIR}/${item}" >&2
        exit 1
    fi
done

# ===== 创建压缩包 =====
echo "正在创建备份压缩包: ${ZIP_PATH}"

cd "${BASE_DIR}"
zip -r "${ZIP_PATH}" "${BACKUP_ITEMS[@]}"

# ===== 上传到 OSS =====
echo "上传备份到 OSS: ${OSS_BACKUP_DIR}/${ZIP_NAME}"

ossutil cp \
    "${ZIP_PATH}" \
    "${OSS_BACKUP_DIR}/${ZIP_NAME}"

# ===== 清理本地临时文件 =====
rm -f "${ZIP_PATH}"

# ===== 备份轮转：删除最旧备份 =====
echo "检查 OSS 备份数量（最大 ${MAX_BACKUPS}）"

mapfile -t BACKUPS < <(
    ossutil ls "${OSS_BACKUP_DIR}/" | \
    awk '{print $NF}' | \
    grep '\.zip$' | \
    sort
)

BACKUP_COUNT="${#BACKUPS[@]}"

if (( BACKUP_COUNT > MAX_BACKUPS )); then
    DELETE_COUNT=$((BACKUP_COUNT - MAX_BACKUPS))
    echo "需要删除 ${DELETE_COUNT} 个旧备份"

    for ((i=0; i<DELETE_COUNT; i++)); do
        OLD_BACKUP="${BACKUPS[$i]}"
        echo "删除旧备份: ${OLD_BACKUP}"
        ossutil rm "${OLD_BACKUP}"
    done
else
    echo "当前备份数量 ${BACKUP_COUNT}，无需删除"
fi

echo "备份完成"