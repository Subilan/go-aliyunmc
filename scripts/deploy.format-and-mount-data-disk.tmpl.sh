#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

echo "格式化并挂载数据盘"

DATA_DISK_SIZE="{{ .DataDiskSize }} GiB"
DATA_DISK=$(fdisk -l | grep "${DATA_DISK_SIZE}" | head -n 1 | awk '{print $2}' | sed 's/://')
DATA_DISK_MOUNT_POINT="${USER_HOME}/server"

mkdir -p "${DATA_DISK_MOUNT_POINT}"
mkfs.ext4 "$DATA_DISK"
DATA_DISK_UUID=$(blkid | grep "$DATA_DISK" | sed 's/UUID=/ /g' | sed 's/"/ /g' | awk '{print $2}')
mount "${DATA_DISK}" "${DATA_DISK_MOUNT_POINT}"
cp /etc/fstab /etc/fstab.bak
echo "UUID=${DATA_DISK_UUID} ${DATA_DISK_MOUNT_POINT} ext4 defaults 0 0" >> /etc/fstab
systemctl daemon-reload
