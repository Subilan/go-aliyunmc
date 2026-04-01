#!/bin/bash
set -euo pipefail

USERNAME="{{ .Username }}"
USER_HOME="/home/${USERNAME}"

echo "安装 ossutil"

curl https://gosspublic.alicdn.com/ossutil/install.sh | bash

echo "[Credentials]" >> "/root/.ossutilconfig"
echo "endpoint=oss-{{ .RegionId }}-internal.aliyuncs.com" >> "/root/.ossutilconfig"
echo "accessKeySecret={{ .AccessKeySecret }}" >> "/root/.ossutilconfig"
echo "accessKeyID={{ .AccessKeyId }}" >> "/root/.ossutilconfig"

cp /root/.ossutilconfig "${USER_HOME}"
