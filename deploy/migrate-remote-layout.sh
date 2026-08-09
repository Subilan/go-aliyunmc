#!/usr/bin/env bash
set -euo pipefail

PROD="${1:-/home/gomc/prod}"

if [ -e "$PROD/current" ] || [ -L "$PROD/current" ]; then
  echo "==> Already migrated: $(readlink "$PROD/current")"
  exit 0
fi

echo "==> Creating initial release from current flat layout"
mkdir -p "$PROD/releases/initial"

for item in configs rbac_model.conf rbac_policy.csv minecraft_en_us.json minecraft_zh_cn.json scripts go-aliyunmc; do
  if [ -e "$PROD/$item" ] && [ ! -L "$PROD/$item" ]; then
    mv "$PROD/$item" "$PROD/releases/initial/$item"
  fi
done

printf '{"layout":"releases/current","initial":true}\n' > "$PROD/releases/initial/deploy-manifest.json"
ln -s releases/initial "$PROD/current"

echo "==> Creating stable symlinks"
ln -sfn current/configs "$PROD/configs"
ln -sfn current/rbac_model.conf "$PROD/rbac_model.conf"
ln -sfn current/rbac_policy.csv "$PROD/rbac_policy.csv"
ln -sfn current/minecraft_en_us.json "$PROD/minecraft_en_us.json"
ln -sfn current/minecraft_zh_cn.json "$PROD/minecraft_zh_cn.json"
ln -sfn current/scripts "$PROD/scripts"
ln -sfn current/go-aliyunmc "$PROD/go-aliyunmc"

chown -R gomc:gomc "$PROD/releases/initial"
echo "==> Migration done"
