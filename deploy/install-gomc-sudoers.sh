#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST="${1:-8.217.241.208}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUDOERS_SRC="${SCRIPT_DIR}/sudoers/go-aliyunmc"

scp -q "${SUDOERS_SRC}" "root@${REMOTE_HOST}:/etc/sudoers.d/go-aliyunmc"
ssh "root@${REMOTE_HOST}" 'chmod 440 /etc/sudoers.d/go-aliyunmc && visudo -c'
echo "==> sudoers installed on ${REMOTE_HOST}"
