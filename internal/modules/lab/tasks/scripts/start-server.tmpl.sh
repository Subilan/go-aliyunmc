#!/usr/bin/env bash

# 执行 Minecraft 启动脚本，并检查启动后的 screen 会话。

set +e

cd "{{ .ArchiveDir }}"
cd_rc=$?
if [ "$cd_rc" -ne 0 ]; then
    printf '[start_server] 进入服务器目录失败，退出码: %s\n' "$cd_rc"
    exit 1
fi

printf '[start_server] 阶段 1/3：执行 start.sh\n'
./start.sh
start_rc=$?
printf '[start_server] start.sh 退出码: %s\n' "$start_rc"
if [ "$start_rc" -ne 0 ]; then
    exit 1
fi

printf '[start_server] 阶段 2/3：等待 0.5 秒\n'
sleep 0.5
sleep_rc=$?
printf '[start_server] sleep 退出码: %s\n' "$sleep_rc"
if [ "$sleep_rc" -ne 0 ]; then
    exit 1
fi

printf '[start_server] 阶段 3/3：检查 screen 会话\n'
screen -S server -Q select . >/dev/null
screen_rc=$?
printf '[start_server] screen 检查退出码: %s\n' "$screen_rc"
if [ "$screen_rc" -ne 0 ]; then
    exit 1
fi

exit 0
