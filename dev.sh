#!/usr/bin/env bash
# 开发模式：同时启动后端（:8787）+ 流量代理（:8788）和前端 next dev（:5173）。
# 前端的 /api 会反向代理到后端。Ctrl-C 一并退出。
#
# 单一二进制（内嵌前端）方式见 README 的「单一二进制」一节，不使用本脚本。
set -euo pipefail
cd "$(dirname "$0")"

# 退出时结束本进程组里的所有子进程（后端 + 前端）。
cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

# 后端（普通 go run，不内嵌前端）。并发 work agent 数量在「系统设置」里配置。
go run ./cmd/artex -addr :8787 -proxy 127.0.0.1:8788 &

# 前端热重载（Next dev server，/api 反向代理到 :8787）。
( cd web && npm run dev ) &

echo "[dev] 后端 :8787 / 代理 :8788 / 前端 http://localhost:5173  (Ctrl-C 退出)"
wait
