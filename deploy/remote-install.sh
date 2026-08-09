#!/usr/bin/env bash
set -euo pipefail

RELEASE_NAME="${1:?usage: remote-install.sh <release-name> [prod-dir]}"
PROD="${2:-/home/gomc/prod}"
STAGING="$PROD/.deploy/$RELEASE_NAME"
RELEASE_DIR="$PROD/releases/$RELEASE_NAME"

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

if [ ! -L "$PROD/current" ]; then
  echo "remote layout is not migrated (expected symlink: $PROD/current)" >&2
  exit 1
fi

for item in configs scripts; do
  if [ ! -L "$PROD/$item" ]; then
    echo "remote layout is not migrated (expected symlink: $PROD/$item)" >&2
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

echo "==> Ensuring stable symlinks"
ln -sfn current/configs "$PROD/configs"
ln -sfn current/rbac_model.conf "$PROD/rbac_model.conf"
ln -sfn current/rbac_policy.csv "$PROD/rbac_policy.csv"
ln -sfn current/minecraft_en_us.json "$PROD/minecraft_en_us.json"
ln -sfn current/minecraft_zh_cn.json "$PROD/minecraft_zh_cn.json"
ln -sfn current/scripts "$PROD/scripts"
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

echo "==> Deployed: $(readlink "$PROD/current")"
