#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

if ! command -v ossutil &>/dev/null; then
  echo "安装 ossutil"
  curl https://gosspublic.alicdn.com/ossutil/install.sh | bash
else
  echo "ossutil 已安装，跳过安装"
fi

# 配置写入必须独立于「是否已安装」的判断：上次部署可能在安装后、
# 写配置前失败，直接重试时必须补上配置。完整重写也避免重复凭据。
echo "写入 ossutil 配置"
umask 077
CONFIG=/root/.ossutilconfig
cat > "${CONFIG}" <<EOF
[Credentials]
endpoint=oss-{{ .RegionId }}-internal.aliyuncs.com
accessKeySecret={{ .AccessKeySecret }}
accessKeyID={{ .AccessKeyId }}
EOF
chmod 600 "${CONFIG}"

cp "${CONFIG}" "${USER_HOME}/.ossutilconfig"
chown "${USERNAME}:${USERNAME}" "${USER_HOME}/.ossutilconfig"
