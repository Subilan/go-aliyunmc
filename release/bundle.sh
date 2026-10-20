#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-${OUT:-$ROOT/dist}}"

APP_VERSION="$(cat "$ROOT/VERSION")"
APP_COMMIT="${APP_COMMIT:-$(git -C "$ROOT" rev-parse --short=7 HEAD 2>/dev/null || echo unknown)}"
APP_BUILD_TIME="${APP_BUILD_TIME:-$(date -u +%FT%TZ)}"

rm -rf "$OUT"
mkdir -p "$OUT/assets/rbac" "$OUT/assets/scripts"

echo "==> Building linux/amd64 binary"
(cd "$ROOT" && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
	-ldflags "-X github.com/Subilan/go-aliyunmc/internal/version.Version=$APP_VERSION \
	-X github.com/Subilan/go-aliyunmc/internal/version.Commit=$APP_COMMIT \
	-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=$APP_BUILD_TIME" \
	-o "$OUT/go-aliyunmc" ./cmd/aliyunmc)

echo "==> Copying release assets"
cp "$ROOT/assets/rbac/model.conf" "$ROOT/assets/rbac/policy.csv" "$OUT/assets/rbac/"
cp -R "$ROOT/assets/scripts/." "$OUT/assets/scripts/"
cp "$ROOT/release/systemd/go-aliyunmc.service" "$OUT/go-aliyunmc.service"
cp "$ROOT/release/remote-install.sh" "$OUT/remote-install.sh"

echo "==> Bundle ready: $OUT"
