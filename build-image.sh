#!/usr/bin/env bash
# ARTEX 中文版 —— 构建镜像并启动。
#
# 为什么需要这个脚本
# ----------------
# `Dockerfile` 是「只负责运行」的镜像，**不在容器里编译前端**：它只把预先编译好的
# Linux 二进制 `COPY dist/<arch>/artex` 放进镜像，而那份二进制里已经 embed 了
# `server/webui/dist`（= `web/out`）。也就是说界面内容是在**宿主机上跑 `next build`
# 时**定下来的。
#
# 所以 `docker compose up -d --build` 单独用是不够的：compose 只会重建 Docker 层，
# 而那一层复制的二进制没变（`COPY dist/...` 命中缓存直接结束）。这个脚本负责前半段
# （前端 → 二进制），并只在需要时才重新构建。
#
# 用法
#   ./build-image.sh              # 构建并启动（已有可用二进制则复用）
#   ./build-image.sh --no-up      # 只准备二进制和镜像，不启动
#   ./build-image.sh --force      # 强制重新构建前端与二进制
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*" >&2; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

# ── 读取 .env，让配置只有一处 ──────────────
load_env(){
  [ -f .env ] || return 0
  set +u                      # 避免 .env 里的空值在 set -u 下报错
  set -a; . ./.env; set +a
  set -u
}

DO_UP=1
FORCE=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --no-up) DO_UP=0; shift ;;
    --force) FORCE=1; shift ;;
    --help|-h) sed -n '2,23p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "无法识别的参数：$1（见 --help）" ;;
  esac
done

load_env
# 本仓库固定简体中文。ARTEX_LOCALE 同时决定服务端向数据库播种的显示名语言，
# 因此与界面 locale 取同一个值，避免「中文界面 + 别的语言 agent 名称」。
LOCALE="zh"

command -v go >/dev/null 2>&1 || die "需要 Go 才能编译要放进镜像的二进制：https://go.dev/dl/"
command -v npm >/dev/null 2>&1 || die "需要 Node.js/npm 才能做前端静态构建"
command -v rsync >/dev/null 2>&1 || die "需要 rsync"

# 容器一定是 Linux，所以不管宿主机是什么系统都按 GOOS=linux 编译。
ARCH="$(go env GOARCH)"
BIN="dist/${ARCH}/artex"
# 记录这份二进制的构建标记。界面内容被写进 embed 的 HTML 里，外面看不出来，
# 所以构建时留个记号，下次运行时对比。
MARKER="dist/${ARCH}/.locale"

need_build=0
reason=""
if [ "$FORCE" = "1" ]; then
  need_build=1; reason="--force"
elif [ ! -f "$BIN" ]; then
  need_build=1; reason="二进制不存在"
elif [ ! -f "$MARKER" ]; then
  need_build=1; reason="没有构建记录（不是用本脚本构建的）"
elif [ "$(cat "$MARKER")" != "$LOCALE" ]; then
  need_build=1; reason="上次构建的语言是 $(cat "$MARKER")"
fi

if [ "$need_build" = "1" ]; then
  info "重新构建（语言 ${LOCALE}，原因：${reason}）"
  info "① 前端静态构建…"
  ( cd web && npm ci --include=dev && NEXT_EXPORT=1 NEXT_PUBLIC_LOCALE="$LOCALE" npx next build )
  info "② 同步到内嵌目录…"
  mkdir -p server/webui/dist
  rsync -a --delete web/out/ server/webui/dist/
  info "③ 编译 Linux/${ARCH} 二进制…"
  mkdir -p "dist/${ARCH}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build \
    -tags embedui -trimpath \
    -ldflags "-s -w -buildid= -X main.version=0.3.15-cn" \
    -o "$BIN" ./cmd/artex
  printf '%s' "$LOCALE" > "$MARKER"
  ok "二进制就绪：${BIN}（语言 ${LOCALE}）"
else
  ok "复用已构建好的二进制（如需强制重建，加 --force）"
fi

if [ "$DO_UP" = "1" ]; then
  info "构建镜像并启动…"
  # 界面 locale 在构建期写进 HTML，服务端看不到，所以服务端播种的显示名
  # （reporter 等）语言要用运行时变量传进去，两边保持一致。
  ARTEX_LOCALE="$LOCALE" docker compose up -d --build
  ok "完成 → http://localhost:8787"
  info "查看日志：docker compose logs -f artex"
fi
