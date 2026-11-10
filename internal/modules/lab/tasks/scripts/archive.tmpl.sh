#!/usr/bin/env bash

set -euo pipefail

LOCAL_DIR="{{ .RemoteArchiveDir }}"
OSS_PATH="{{ .OSSRoot }}/{{ .ArchiveName }}"

J=50
P=10

if [ ! -d "${LOCAL_DIR}" ]; then
    echo "ERROR: 本地目录不存在: ${LOCAL_DIR}" >&2
    exit 1
fi

echo "增量同步归档 -> ${OSS_PATH}"

ossutil sync "${LOCAL_DIR}/" "${OSS_PATH}/" \
    --delete \
    --force \
    --update \
    --jobs=$J \
    --parallel=$P > /dev/null

echo "归档完成"

