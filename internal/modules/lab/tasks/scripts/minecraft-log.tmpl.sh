#!/usr/bin/env bash

# 读取本次 Minecraft 启动产生的 latest.log 尾部内容。

set +e

log_file="{{ .LogPath }}"
if [ ! -f "$log_file" ]; then
    printf 'Minecraft 日志文件不存在: %s\n' "$log_file"
    exit 0
fi

log_size=$(stat -c %s "$log_file" 2>/dev/null || echo 0)
start_offset={{ .Offset }}
if [ "$log_size" -lt "$start_offset" ]; then
    start_offset=0
fi

tail -c "+$((start_offset + 1))" "$log_file" | tail -c {{ .MaxBytes }} | tail -n {{ .MaxLines }}

exit 0
