#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST="${1:-8.217.241.208}"
REMOTE_USER="${2:-gomc}"
APP_HOST="${REMOTE_USER}@${REMOTE_HOST}"
ROOT_HOST="root@${REMOTE_HOST}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UNIT_SRC="${SCRIPT_DIR}/systemd/go-aliyunmc.service"

echo "==> Uploading systemd unit to ${ROOT_HOST}"
scp -q "${UNIT_SRC}" "${ROOT_HOST}:/tmp/go-aliyunmc.service"

echo "==> Installing and enabling systemd unit"
ssh "${ROOT_HOST}" 'install -m 0644 /tmp/go-aliyunmc.service /etc/systemd/system/go-aliyunmc.service && rm -f /tmp/go-aliyunmc.service && systemctl daemon-reload && systemctl enable go-aliyunmc.service'

echo "==> Stopping old screen session"
ssh "${APP_HOST}" 'screen -S gomc -X quit 2>/dev/null || true
for _ in $(seq 1 40); do
  if ! pgrep -f "[g]o-aliyunmc run" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done'

echo "==> Starting systemd service"
if ! ssh "${ROOT_HOST}" 'systemctl start go-aliyunmc.service'; then
  echo "!!> systemd start failed, restoring screen session" >&2
  ssh "${APP_HOST}" 'cd /home/gomc/prod && ./start.sh' || true
  exit 1
fi

echo "==> Verifying service"
ssh "${ROOT_HOST}" 'systemctl is-active go-aliyunmc.service && systemctl status go-aliyunmc.service --no-pager -n 12'
ssh "${APP_HOST}" 'ps -o pid,user,etime,cmd -C go-aliyunmc || true'
