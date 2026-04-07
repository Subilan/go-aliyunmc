#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${HURL_ENV_FILE:-$SCRIPT_DIR/../env/dev.env}"
HURL_BIN="${HURL_BIN:-hurl}"

if [[ $# -lt 1 ]]; then
  echo "用法: $0 <hurl-file> [hurl-options...]"
  echo "示例: $0 ./create-instance.hurl --variable instance=ecs.e-c1m2.large --variable zone=cn-shenzhen-f"
  exit 1
fi

if ! command -v "$HURL_BIN" >/dev/null 2>&1; then
  echo "未找到 hurl 命令: $HURL_BIN"
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "未找到 jq 命令，请先安装 jq"
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

HURL_FILE="$1"
shift

if [[ ! -f "$HURL_FILE" ]]; then
  echo "hurl 文件不存在: $HURL_FILE"
  exit 1
fi

BASE_URL="$(awk -F'=' '/^baseURL=/{print $2; exit}' "$ENV_FILE" | tr -d '\r')"
if [[ -z "$BASE_URL" ]]; then
  echo "无法从 $ENV_FILE 读取 baseURL"
  exit 1
fi

stream_sse() {
  local task_id="$1"
  local sse_url="${BASE_URL%/}/task/${task_id}/output"
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

RESP_FILE="$(mktemp)"
BODY_FILE="$(mktemp)"
cleanup() {
  rm -f "$RESP_FILE"
  rm -f "$BODY_FILE"
}
trap cleanup EXIT

echo "执行 hurl: $HURL_FILE"
"$HURL_BIN" --variables-file "$ENV_FILE" "$HURL_FILE" "$@" --no-color --no-pretty --include --output "$RESP_FILE"

# 从响应首行解析 HTTP 状态码
STATUS_CODE="$(sed -nE '1{s#^HTTP/[0-9.]+ ([0-9]{3}).*#\1#;p;q;}' "$RESP_FILE")"

# 去掉响应头，仅保留响应体
awk 'BEGIN{body=0} {line=$0; sub(/\r$/, "", line)} body {print line} line=="" {body=1}' "$RESP_FILE" > "$BODY_FILE"

if [[ "$STATUS_CODE" != "200" ]]; then
  echo "请求未返回200，状态码: ${STATUS_CODE:-unknown}"
  echo "响应内容如下:"
  cat "$BODY_FILE"
  exit 1
fi

TASK_ID=""
TASK_ID="$(jq -r '.data.ID // .data.id // .ID // .id // empty' "$BODY_FILE" 2>/dev/null || true)"

# 去除可能的空白/回车，避免误判为空。
TASK_ID="$(printf '%s' "$TASK_ID" | tr -d '\r[:space:]')"

if [[ -z "$TASK_ID" ]]; then
  echo "未能从响应中解析任务ID，响应内容如下:"
  cat "$BODY_FILE"
  exit 1
fi

echo "任务ID: $TASK_ID"
stream_sse "$TASK_ID"
