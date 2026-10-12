#!/usr/bin/env bash
set -euo pipefail

# 1. 安装前端（Next.js）依赖并做静态构建
echo "[*] Building Frontend..."
cd web
npm ci
npm run build:static
cd ..

# 2. 把前端构建产物复制到 Go 可以内嵌（embed）的位置
echo "[*] Copying static assets..."
mkdir -p server/webui/dist
cp -r web/out/* server/webui/dist/

# 3. 编译 Go 单一二进制（必须带 embedui 标签）
echo "[*] Compiling Go Backend..."
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex

echo "[+] Build Completed successfully!"
