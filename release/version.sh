#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT/VERSION"
SEMVER='^[0-9]+\.[0-9]+\.[0-9]+$'

PUSH=0
DRY_RUN=0
TARGET=""

usage() {
	cat <<'EOF'
用法: release/version.sh <major|minor|patch|X.Y.Z> [--push] [--dry-run]

按 semver 更新 VERSION，提交后打 v<版本号> 注解标签。
EOF
}

for arg in "$@"; do
	case "$arg" in
	--push) PUSH=1 ;;
	--dry-run) DRY_RUN=1 ;;
	-h | --help)
		usage
		exit 0
		;;
	-*)
		echo "未知选项: $arg" >&2
		usage >&2
		exit 1
		;;
	*)
		if [[ -n "$TARGET" ]]; then
			echo "只接受一个版本参数" >&2
			exit 1
		fi
		TARGET="$arg"
		;;
	esac
done

if [[ -z "$TARGET" ]]; then
	usage >&2
	exit 1
fi

CURRENT="$(tr -d '[:space:]' <"$VERSION_FILE")"
if [[ ! "$CURRENT" =~ $SEMVER ]]; then
	echo "VERSION 内容不是 semver: $CURRENT" >&2
	exit 1
fi

next_version() {
	local target="$1"
	if [[ "$target" =~ $SEMVER ]]; then
		echo "$target"
		return
	fi
	local major minor patch
	IFS=. read -r major minor patch <<<"$CURRENT"
	case "$target" in
	major)
		major=$((major + 1))
		minor=0
		patch=0
		;;
	minor)
		minor=$((minor + 1))
		patch=0
		;;
	patch) patch=$((patch + 1)) ;;
	*)
		echo "无法识别的升级类型: $target" >&2
		exit 1
		;;
	esac
	echo "$major.$minor.$patch"
}

NEXT="$(next_version "$TARGET")"
TAG="v$NEXT"

if git -C "$ROOT" rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
	echo "标签已存在: $TAG" >&2
	exit 1
fi

if [[ "$NEXT" == "$CURRENT" ]]; then
	echo "版本未变化: $CURRENT" >&2
	exit 1
fi

if ! git -C "$ROOT" diff --quiet HEAD -- VERSION; then
	echo "VERSION 有未提交的改动" >&2
	exit 1
fi

echo "VERSION: $CURRENT -> $NEXT"

if [[ "$DRY_RUN" == 1 ]]; then
	echo "commit: chore(release): $TAG"
	echo "tag: $TAG"
	if [[ "$PUSH" == 1 ]]; then
		echo "push: origin HEAD, $TAG"
	fi
	exit 0
fi

printf '%s\n' "$NEXT" >"$VERSION_FILE"
git -C "$ROOT" add VERSION
git -C "$ROOT" commit -m "chore(release): $TAG"
git -C "$ROOT" tag -a "$TAG" -m "$TAG"

if [[ "$PUSH" == 1 ]]; then
	git -C "$ROOT" push origin HEAD
	git -C "$ROOT" push origin "$TAG"
fi

echo "已发布 $TAG"
