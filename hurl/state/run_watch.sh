#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${HURL_ENV_FILE:-$SCRIPT_DIR/../env/dev.env}"
WATCH_PATH="${1:-}"

if [[ -z "$WATCH_PATH" ]]; then
  echo "用法: $0 <watch-path>"
  echo "示例: $0 /state/watch/server-status"
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "未找到 curl 命令，请先安装 curl"
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "变量文件不存在: $ENV_FILE"
  exit 1
fi

BASE_URL="$(awk -F'=' '/^baseURL=/{print $2; exit}' "$ENV_FILE" | tr -d '\r')"
if [[ -z "$BASE_URL" ]]; then
  echo "无法从 $ENV_FILE 读取 baseURL"
  exit 1
fi

WATCH_URL="${BASE_URL%/}${WATCH_PATH}"

stream_sse() {
  local sse_url="$1"
  local event_name="message"
  local data_lines=""

  echo "开始监听SSE: $sse_url"

  # 连接关闭后，while 循环结束，函数返回。
  curl -sS -N --no-buffer -H "Accept: text/event-stream" "$sse_url" | while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"

    if [[ -z "$line" ]]; then
      if [[ -n "$data_lines" ]]; then
        local now
        now="$(date +"%H:%M:%S")"
        printf '[%s] %s: %s\n' "$now" "$event_name" "$data_lines"
      fi
      event_name="message"
      data_lines=""
      continue
    fi

    if [[ "$line" == event:* ]]; then
      event_name="${line#event: }"
      continue
    fi

    if [[ "$line" == data:* ]]; then
      local payload
      payload="${line#data: }"
      if [[ -z "$data_lines" ]]; then
        data_lines="$payload"
      else
        data_lines+="\\n$payload"
      fi
    fi
  done

  echo "SSE连接已关闭，退出监听"
}

stream_sse "$WATCH_URL"
