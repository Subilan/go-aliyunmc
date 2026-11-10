#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"
MOUNT_POINT="${USER_HOME}/server"
EXPECTED_BYTES=$(( {{ .DataDiskSize }} * 1024 * 1024 * 1024 ))
STATE_DIR="/var/lib/aliyunmc"
MARKER="${STATE_DIR}/data-disk-uuid"

echo "检查数据盘"

mkdir -p "${MOUNT_POINT}" "${STATE_DIR}"

if mountpoint -q "${MOUNT_POINT}" 2>/dev/null; then
  echo "数据盘已挂载，跳过"
  exit 0
fi

# 找到根盘并排除它，避免只按容量匹配时误选系统盘。
ROOT_SOURCE=$(findmnt -n -o SOURCE /)
ROOT_DISK=$(lsblk -no PKNAME "${ROOT_SOURCE}" 2>/dev/null | head -n 1)
if [ -z "${ROOT_DISK}" ]; then
  ROOT_DISK=$(basename "${ROOT_SOURCE}")
fi

# 在未挂载的整盘中按配置容量匹配数据盘，必须唯一命中。
CANDIDATES=()
while read -r name size type; do
  [ "${type}" = "disk" ] || continue
  [ "${name}" = "${ROOT_DISK}" ] && continue
  [ "${size}" = "${EXPECTED_BYTES}" ] || continue
  if lsblk -rno MOUNTPOINT "/dev/${name}" | grep -q '^/'; then
    continue
  fi
  CANDIDATES+=("/dev/${name}")
done < <(lsblk -bdn -o NAME,SIZE,TYPE)

if [ "${#CANDIDATES[@]}" -ne 1 ]; then
  echo "无法唯一确定数据盘：期望 ${EXPECTED_BYTES} 字节，找到 ${#CANDIDATES[@]} 个候选：${CANDIDATES[*]:-}"
  exit 1
fi

DATA_DISK="${CANDIDATES[0]}"
echo "数据盘：${DATA_DISK}"

EXISTING_FS=$(blkid -o value -s TYPE "${DATA_DISK}" 2>/dev/null || true)
DATA_DISK_UUID=$(blkid -o value -s UUID "${DATA_DISK}" 2>/dev/null || true)
MARKED_UUID=""
[ -f "${MARKER}" ] && MARKED_UUID=$(cat "${MARKER}")

# 兼容旧版本部署：旧脚本会格式化并写入 fstab，但没有 marker。
# fstab 中已有本挂载点时，认为这块盘就是本系统格式化的。
if [ -z "${MARKED_UUID}" ] && grep -q "[[:space:]]${MOUNT_POINT}[[:space:]]" /etc/fstab; then
  MARKED_UUID="${DATA_DISK_UUID}"
fi

if [ -n "${EXISTING_FS}" ]; then
  if [ -z "${MARKED_UUID}" ] || [ "${DATA_DISK_UUID}" != "${MARKED_UUID}" ]; then
    echo "数据盘 ${DATA_DISK} 已存在文件系统（${EXISTING_FS}，UUID=${DATA_DISK_UUID}），但没有本系统写入的标记，拒绝格式化"
    exit 1
  fi
  echo "数据盘已由本系统格式化过，直接挂载"
else
  # 已有分区表的磁盘同样拒绝处理，避免误伤。
  if lsblk -nrno NAME "${DATA_DISK}" | sed '1d' | grep -q .; then
    echo "数据盘 ${DATA_DISK} 存在分区，拒绝格式化"
    exit 1
  fi

  echo "格式化数据盘"
  mkfs.ext4 "${DATA_DISK}"
  DATA_DISK_UUID=$(blkid -o value -s UUID "${DATA_DISK}")
  if [ -z "${DATA_DISK_UUID}" ]; then
    echo "无法读取数据盘 UUID，停止部署"
    exit 1
  fi
  echo "${DATA_DISK_UUID}" > "${MARKER}"
fi

echo "挂载数据盘"
mount "${DATA_DISK}" "${MOUNT_POINT}"

# 幂等写入 fstab：先移除同一挂载点的旧条目，再追加当前条目。
cp -n /etc/fstab /etc/fstab.aliyunmc.bak
grep -v "[[:space:]]${MOUNT_POINT}[[:space:]]" /etc/fstab > /etc/fstab.aliyunmc.new || true
echo "UUID=${DATA_DISK_UUID} ${MOUNT_POINT} ext4 defaults 0 0" >> /etc/fstab.aliyunmc.new
mv /etc/fstab.aliyunmc.new /etc/fstab
systemctl daemon-reload
