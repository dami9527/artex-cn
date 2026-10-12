#!/usr/bin/env bash
set -euo pipefail

# Render 会自行分配 PORT 环境变量，因此显式把该端口作为监听地址传给应用。
# 保持默认的 skill/数据目录不变。
: "${PORT:?必须设置 PORT 环境变量}"
echo "[*] Starting ARTEX Server on port :$PORT"
exec ./artex -addr ":$PORT"
