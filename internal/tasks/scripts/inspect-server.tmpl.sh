#!/usr/bin/env bash

# 输出 Minecraft 服务器、screen 会话、端口占用和日志文件状态。

set +e

if command -v screen >/dev/null 2>&1 && screen -S server -Q select . >/dev/null 2>&1; then
    printf 'screen=1\n'
else
    printf 'screen=0\n'
fi

if command -v pgrep >/dev/null 2>&1 && pgrep -af '[j]ava .*server[.]jar' >/dev/null 2>&1; then
    printf 'java=1\n'
else
    printf 'java=0\n'
fi

port_line=$(ss -Hltnp "sport = :{{ .Port }}" 2>/dev/null | head -n 1)
if [ -n "$port_line" ]; then
    printf 'port=1\n'
    printf 'owner=%s\n' "$port_line"
else
    printf 'port=0\n'
fi

log_size=$(stat -c %s "{{ .LogPath }}" 2>/dev/null || echo 0)
printf 'log_size=%s\n' "$log_size"

exit 0
