#!/usr/bin/env bash
# ARTEX 更新脚本：① Docker 更新（用当前源码重新构建镜像并重建容器）  ② 本地编译更新（重新编译二进制）
# 与 install.sh 配对：install 负责首次安装，update 负责升级到新版本。
# 不需要单独跑数据库迁移：artex 每次启动都会幂等地重跑 schema.sql（包含 ADD COLUMN、
# CREATE INDEX IF NOT EXISTS），所以「重启即迁移」。数据（pgdata 卷、./data、./skills）不受影响。
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }

# 如果存在 .env 就先读入，把配置（POSTGRES_PASSWORD、NEXT_PUBLIC_LOCALE 等）集中到一处。
load_env(){
  [ -f .env ] || return 0
  set +u                      # 避免 .env 里的空值在 set -u 下报错
  set -a; . ./.env; set +a
  set -u
}

# ── 可选：把仓库同步到最新代码（compose、脚本、本地编译源码都会随之更新） ───────
sync_repo(){
  [ -d .git ] && command -v git >/dev/null 2>&1 || { warn "这不是 git 工作副本，跳过 git pull"; return; }
  [ "$(ask '要拉取最新代码吗（git pull --ff-only）？(y/n)' y)" = y ] || return
  if ! git pull --ff-only; then
    warn "git pull 无法快进（本地有改动或分支已分叉）。请自行处理后重试，本次沿用当前代码"
  fi
}

# ── ① Docker 更新 ───────────────────────────────
# 本仓库不发布镜像（上游 autumn27/artex 已从 Docker Hub 下架）。
# 所以「更新」不是拉新镜像，而是用当前源码重新构建镜像。
update_docker(){
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 \
    || die "找不到 docker / docker compose。请先用 ./install.sh 安装部署"
  [ -f .env ] || die "找不到 .env。请先用 ./install.sh 完成首次部署"
  [ -x ./build-image.sh ] || die "找不到 build-image.sh（需要可执行权限）"

  # 构建（前端 → 内嵌 → Linux 二进制）与语言判定都交给 build-image.sh。
  # 语言没变就不重建；变了就用新语言重建后再启动。
  ./build-image.sh
}

# ── ② 本地编译更新 ──────────────────────────────
update_local(){
  command -v go >/dev/null 2>&1 || die "找不到 Go（>=1.26）：https://go.dev/dl/"
  [ -f config.json ] || warn "找不到 config.json。如果是首次部署，请使用 ./install.sh"
  ok "Go: $(go version)"

  if command -v npm >/dev/null 2>&1; then
    load_env
    info "重新构建前端静态产物…（界面语言：${NEXT_PUBLIC_LOCALE:-zh}）"
    ( cd web && npm ci && NEXT_EXPORT=1 NEXT_PUBLIC_LOCALE="${NEXT_PUBLIC_LOCALE:-zh}" npm run build:static )
    rm -rf server/webui/dist && cp -r web/out server/webui/dist
    info "重新编译内嵌前端的单一二进制…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "找不到 npm：只编译不内嵌前端的后端（前端需要用 npm run dev 单独启动）"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "编译完成 → ./artex"
  warn "要让改动生效，请重启正在运行的 artex 进程（重启时会自动迁移 schema）"
}

echo "=============================="
echo "  ARTEX 更新"
echo "  1) Docker 更新（重新构建镜像并重建容器）"
echo "  2) 本地更新（用 go 重新编译）"
echo "=============================="
case "$(ask '选择' 1)" in
  1) sync_repo; update_docker ;;
  2) sync_repo; update_local ;;
  *) die "选择无效" ;;
esac
