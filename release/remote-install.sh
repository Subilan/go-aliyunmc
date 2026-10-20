#!/usr/bin/env bash
set -euo pipefail

RELEASE_NAME="${1:?usage: remote-install.sh <release-name> [prod-dir]}"
PROD="${2:-/home/gomc/prod}"
STAGING="$PROD/.deploy/$RELEASE_NAME"
RELEASE_DIR="$PROD/releases/$RELEASE_NAME"

# 保留的 release 总数（含 current）。回滚只需要 current 的前一个，3 是安全下限。
KEEP_RELEASES="${KEEP_RELEASES:-3}"

if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
else
  SUDO="sudo"
fi

case "$RELEASE_NAME" in
  */*|*..*) echo "invalid release name: $RELEASE_NAME" >&2; exit 1 ;;
esac

if [ ! -f "$STAGING/SHA256SUMS" ]; then
  echo "staging bundle not found: $STAGING" >&2
  exit 1
fi

for item in config assets; do
  if [ -e "$PROD/$item" ] && [ ! -L "$PROD/$item" ]; then
    echo "unexpected $PROD/$item: expected a symlink" >&2
    exit 1
  fi
done

echo "==> Verifying checksums"
(cd "$STAGING" && sha256sum -c SHA256SUMS)

PREV=""
if [ -L "$PROD/current" ]; then
  PREV="$(readlink "$PROD/current")"
fi

if [ -e "$RELEASE_DIR" ]; then
  echo "release already exists: $RELEASE_DIR" >&2
  exit 1
fi

echo "==> Installing release"
mkdir -p "$PROD/releases"
mv "$STAGING" "$RELEASE_DIR"
ln -sfn "releases/$RELEASE_NAME" "$PROD/current"
mkdir -p "$PROD/state/logs" "$PROD/state/cache"
if [ "$(id -u)" -eq 0 ]; then
  chown -R gomc:gomc "$PROD/state"
fi
ln -sfn current/config "$PROD/config"
ln -sfn current/assets "$PROD/assets"
ln -sfn current/go-aliyunmc "$PROD/go-aliyunmc"

echo "==> Installing systemd unit"
$SUDO install -m 0644 "$RELEASE_DIR/go-aliyunmc.service" /etc/systemd/system/go-aliyunmc.service
$SUDO systemctl daemon-reload

if [ "$(id -u)" -eq 0 ]; then
  chown -R gomc:gomc "$RELEASE_DIR"
fi

echo "==> Restarting service"
$SUDO systemctl restart go-aliyunmc.service

sleep 2
if ! systemctl is-active --quiet go-aliyunmc; then
  echo "!!> service failed to start" >&2
  if [ -n "$PREV" ] && [ -d "$PROD/$PREV" ]; then
    echo "==> Rolling back to $PREV"
    ln -sfn "$PREV" "$PROD/current"
    $SUDO systemctl restart go-aliyunmc.service || true
  fi
  exit 1
fi

echo "==> Health check"
if ! curl -k -fsS -o /dev/null --max-time 15 https://127.0.0.1/; then
  echo "!!> health check failed" >&2
  if [ -n "$PREV" ] && [ -d "$PROD/$PREV" ]; then
    ln -sfn "$PREV" "$PROD/current"
    $SUDO systemctl restart go-aliyunmc.service || true
  fi
  exit 1
fi

echo "==> Pruning old releases (keeping $KEEP_RELEASES)"
CURRENT_NAME="$(basename "$(readlink "$PROD/current")")"
OLD_RELEASES="$(
  ls -1t "$PROD/releases" 2>/dev/null \
    | tail -n +$((KEEP_RELEASES + 1)) \
    | grep -vx -- "$CURRENT_NAME" || true
)"
if [ -n "$OLD_RELEASES" ]; then
  while IFS= read -r name; do
    case "$name" in
      ""|*/*|*..*) echo "    skip suspicious release name: $name" >&2; continue ;;
    esac
    echo "    removing releases/$name"
    rm -rf "${PROD:?}/releases/${name:?}"
  done <<< "$OLD_RELEASES"
fi

# 只清理滞留超过一天的暂存目录，避免误删正在进行中的发布。
find "$PROD/.deploy" -mindepth 1 -maxdepth 1 -type d -mtime +1 -exec rm -rf {} + 2>/dev/null || true

echo "==> Deployed: $(readlink "$PROD/current")"
