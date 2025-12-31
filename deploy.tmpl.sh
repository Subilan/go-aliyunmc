#!/bin/bash
set -euo pipefail
export MAN_DISABLE_CACHE=yes
export DEBIAN_FRONTEND=noninteractive

dpkg-divert --local --rename --add /var/lib/dpkg/info/man-db.triggers || true

USERNAME="{{ .Username }}"
PASSWORD="{{ .Password }}"
USER_HOME="/home/${USERNAME}"

SSH_PUBLIC_KEY="{{ .SSHPublicKey }}"

echo "1. 创建用户"

if ! id "${USERNAME}" &>/dev/null; then
    useradd -m -s /bin/bash "${USERNAME}"
fi

echo "${USERNAME}:${PASSWORD}" | chpasswd

mkdir -p "${USER_HOME}"
chown -R "${USERNAME}:${USERNAME}" "${USER_HOME}"
chmod 700 "${USER_HOME}"

echo "2. 配置 SSH authorized_keys"

SSH_DIR="${USER_HOME}/.ssh"
AUTHORIZED_KEYS="${SSH_DIR}/authorized_keys"

mkdir -p "${SSH_DIR}"
chmod 700 "${SSH_DIR}"

# 写入公钥（避免重复）
if ! grep -qxF "${SSH_PUBLIC_KEY}" "${AUTHORIZED_KEYS}" 2>/dev/null; then
    echo "${SSH_PUBLIC_KEY}" >> "${AUTHORIZED_KEYS}"
fi

chown -R "${USERNAME}:${USERNAME}" "${SSH_DIR}"
chmod 600 "${AUTHORIZED_KEYS}"

echo "3. 配置源"

echo "# 默认注释了源码镜像以提高 apt update 速度，如有需要可自行取消注释
deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free non-free-firmware
# deb-src https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free non-free-firmware

deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free non-free-firmware
# deb-src https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free non-free-firmware

deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-backports main contrib non-free non-free-firmware
# deb-src https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-backports main contrib non-free non-free-firmware

# 以下安全更新软件源包含了官方源与镜像站配置，如有需要可自行修改注释切换
deb https://mirrors.tuna.tsinghua.edu.cn/debian-security bookworm-security main contrib non-free non-free-firmware
# deb-src https://mirrors.tuna.tsinghua.edu.cn/debian-security bookworm-security main contrib non-free non-free-firmware" > /etc/apt/sources.list

apt update -y

echo "4. 配置 Zulu Java 源"
apt install -y gnupg ca-certificates curl

curl -s https://repos.azul.com/azul-repo.key \
  | sudo gpg --dearmor -o /usr/share/keyrings/azul.gpg

echo "deb [signed-by=/usr/share/keyrings/azul.gpg] https://repos.azul.com/zulu/deb stable main" \
  | sudo tee /etc/apt/sources.list.d/zulu.list
  
chmod 644 /usr/share/keyrings/azul.gpg

apt update -y

echo "5. 安装系统软件"

apt install -y zulu{{ .JavaVersion }}-jre-headless {{ range .Packages }}{{ . }} {{ end }}

echo "6. 安装 ossutil"

curl https://gosspublic.alicdn.com/ossutil/install.sh | bash

echo "[Credentials]" >> "${USER_HOME}/.ossutilconfig"
echo "endpoint=oss-{{ .RegionId }}-internal.aliyuncs.com" >> "${USER_HOME}/.ossutilconfig"
echo "accessKeySecret={{ .AccessKeySecret }}" >> "${USER_HOME}/.ossutilconfig"
echo "accessKeyID={{ .AccessKeyId }}" >> "${USER_HOME}/.ossutilconfig"

echo "7. 挂载数据盘"

DATA_DISK_SIZE="{{ .DataDiskSize }} GiB"
DATA_DISK=$(fdisk -l | grep "${DATA_DISK_SIZE}" | head -n 1 | awk '{print $2}' | sed 's/://')
DATA_DISK_MOUNT_POINT="${USER_HOME}/server"

mkdir -p "${DATA_DISK_MOUNT_POINT}"
mkfs.ext4 $DATA_DISK
DATA_DISK_UUID=`blkid | grep $DATA_DISK | sed 's/UUID=/ /g' | sed 's/"/ /g' | awk '{print $2}'`
mount "${DATA_DISK}" "${DATA_DISK_MOUNT_POINT}"
cp /etc/fstab /etc/fstab.bak
echo "UUID=${DATA_DISK_UUID} ${DATA_DISK_MOUNT_POINT} ext4 defaults 0 0"
echo "UUID=${DATA_DISK_UUID} ${DATA_DISK_MOUNT_POINT} ext4 defaults 0 0" >> /etc/fstab
systemctl daemon-reload

echo "部署成功完成"