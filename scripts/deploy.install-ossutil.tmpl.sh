#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

if command -v ossutil &>/dev/null; then
  echo "ossutil 已安装，跳过"
  exit 0
fi

echo "安装 ossutil"
curl https://gosspublic.alicdn.com/ossutil/install.sh | bash

echo "[Credentials]" >> "/root/.ossutilconfig"
echo "endpoint=oss-{{ .RegionId }}-internal.aliyuncs.com" >> "/root/.ossutilconfig"
echo "accessKeySecret={{ .AccessKeySecret }}" >> "/root/.ossutilconfig"
echo "accessKeyID={{ .AccessKeyId }}" >> "/root/.ossutilconfig"

cp /root/.ossutilconfig "${USER_HOME}"
