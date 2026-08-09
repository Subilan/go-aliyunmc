#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT/dist}"
REF="${GITHUB_SHA:-$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo unknown)}"
REF_SHORT="$(printf '%s' "$REF" | cut -c1-7)"
APP_VERSION="$(cat "$ROOT/VERSION")"
APP_BUILD_TIME="$(date -u +%FT%TZ)"

mkdir -p "$OUT"

echo "==> Building linux/amd64 binary"
(cd "$ROOT" && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
	-ldflags "-X github.com/Subilan/go-aliyunmc/internal/version.Version=$APP_VERSION \
	-X github.com/Subilan/go-aliyunmc/internal/version.Commit=$REF_SHORT \
	-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=$APP_BUILD_TIME" \
	-o "$OUT/go-aliyunmc" .)

echo "==> Copying app assets"
cp "$ROOT/rbac_model.conf" "$ROOT/rbac_policy.csv" "$ROOT/minecraft_en_us.json" "$ROOT/minecraft_zh_cn.json" "$OUT/"
cp -R "$ROOT/scripts" "$OUT/scripts"
cp "$ROOT/deploy/systemd/go-aliyunmc.service" "$OUT/go-aliyunmc.service"
cp "$ROOT/deploy/remote-install.sh" "$OUT/remote-install.sh"

printf '{"app_ref":"%s","version":"%s","built_at":"%s"}\n' "$REF" "$APP_VERSION" "$APP_BUILD_TIME" > "$OUT/deploy-manifest.json"

echo "==> Writing checksums"
(cd "$OUT" && {
  sha256sum go-aliyunmc rbac_model.conf rbac_policy.csv \
    minecraft_en_us.json minecraft_zh_cn.json \
    go-aliyunmc.service remote-install.sh
  find scripts -type f | sort | xargs sha256sum
} > SHA256SUMS)

echo "==> Public bundle ready: $OUT"
