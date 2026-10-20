#!/usr/bin/env bash
set -uo pipefail

root=$(cd "$(dirname "$0")/../../../.." && pwd)
docs="$root/docs"
status=0

fail() {
  printf 'ERROR: %s\n' "$1" >&2
  status=1
}

if [[ ! -d "$docs" ]]; then
  fail "docs/ 不存在"
  exit 1
fi

shopt -s nullglob
files=("$docs"/*.md)
if [[ ${#files[@]} -eq 0 ]]; then
  fail "docs/ 下没有 Markdown 文档"
fi

for file in "${files[@]}"; do
  name=${file##*/}
  if [[ $(head -n 1 "$file") != "---" ]]; then
    fail "$name 缺少 YAML frontmatter"
    continue
  fi

  fm=$(awk 'NR == 1 { next } /^---$/ { exit } { print }' "$file")
  title=$(printf '%s\n' "$fm" | sed -n 's/^title:[[:space:]]*//p' | head -n 1)
  desc=$(printf '%s\n' "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -n 1)

  [[ -n $title ]] || fail "$name 缺少 title"
  [[ -n $desc ]] || fail "$name 缺少 description"

  if [[ -n $desc ]]; then
    desc_len=$(printf '%s' "$desc" | wc -m | tr -d ' ')
    if [[ $desc_len -gt 60 ]]; then
      fail "$name 的 description 有 $desc_len 字，上限 60"
    fi
  fi

  if [[ $name != README.md ]] && ! grep -q "($name)" "$docs/README.md"; then
    fail "docs/README.md 未索引 $name"
  fi
done

for target in $(grep -o '](\([A-Za-z0-9._-]*\.md\))' "$docs/README.md" | sed 's/^](//; s/)$//'); do
  [[ -f "$docs/$target" ]] || fail "docs/README.md 指向不存在的 $target"
done

if [[ $status -eq 0 ]]; then
  printf 'docs 校验通过（%d 篇）\n' "${#files[@]}"
fi
exit $status
