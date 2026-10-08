#!/usr/bin/env bash
# ARTEX 安装脚本：① 全部用 Docker  ② 本地编译运行
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
rand(){ head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24; }

# ── Docker 环境检测与自动安装 ──────────────────
ensure_docker(){
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    ok "已检测到 docker 与 docker compose"; return
  fi
  warn "找不到 docker / docker compose"
  case "$(uname -s)" in
    Linux)
      if [ "$(ask '要自动安装 Docker 吗？(y/n)' y)" = y ]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER" || true
        ok "Docker 安装完成（用户组变更需要重新登录后才会免 sudo 生效）"
      else
        die "请自行安装 docker 后重新运行"
      fi ;;
    Darwin) die "macOS 请安装 Docker Desktop：https://www.docker.com/products/docker-desktop/" ;;
    *)      die "请自行安装 docker 后重新运行" ;;
  esac
}

# ── ① 全部用 Docker ──────────────────────────────
# Dockerfile 是「只负责运行」的镜像，要求事先编译好 Linux 二进制（dist/<arch>/artex）。
# 上游镜像 autumn27/artex 已从 Docker Hub 下架，无法用 pull 启动。
# 构建逻辑（前端 → 内嵌 → Linux 二进制）只放在 build-image.sh 一处。
# 语言判定、是否需要重建、读取 .env 都由那个脚本负责，这里只做委托
# （同样的逻辑写两份会让界面语言悄悄跑偏）。
build_artex_image(){
  [ -x ./build-image.sh ] || die "找不到 build-image.sh（需要可执行权限）"
  ./build-image.sh --no-up
}

install_docker(){
  ensure_docker
  if [ ! -f .env ]; then
    cp .env.example .env 2>/dev/null || true
    local pw key
    pw="$(ask 'Postgres 密码（直接回车则随机生成）' "$(rand)")"
    key="$(ask 'ANTHROPIC_API_KEY（可以留空，稍后在界面里配置）' '')"
    sed -i.bak "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${pw}|" .env
    sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${key}|" .env
    rm -f .env.bak
    ok "已生成 .env 文件（POSTGRES_PASSWORD 已设置）"
  else
    info "沿用已有的 .env 文件"
  fi
  build_artex_image
  info "构建镜像并启动…"
  docker compose up -d --build
  ok "启动完成 → http://localhost:8787（界面语言：${NEXT_PUBLIC_LOCALE:-zh}）"
  info "查看日志：docker compose logs -f artex"
}

# ── ② 本地编译运行 ────────────────────────────
install_local(){
  echo "数据库安装方式："
  echo "  1) 连接已有的 PostgreSQL"
  echo "  2) 用 Docker 起一个 PostgreSQL（需要 docker）"
  case "$(ask '选择' 1)" in
    2)
      ensure_docker
      local pw; pw="$(ask 'Postgres 密码（直接回车则随机生成）' "$(rand)")"
      docker run -d --name artex-pg -p 5432:5432 \
        -e POSTGRES_USER=artex -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=artex \
        -v artex-pg:/var/lib/postgresql/data postgres:16-alpine
      DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=artex DB_PASS="$pw" DB_NAME=artex DB_SSL=disable ;;
    *)
      DB_HOST="$(ask '数据库地址' 127.0.0.1)"
      DB_PORT="$(ask '端口' 5432)"
      DB_USER="$(ask '账号' artex)"
      DB_PASS="$(ask '密码' '')"
      DB_NAME="$(ask '数据库名' artex)"
      DB_SSL="$(ask 'sslmode (disable/require)' disable)" ;;
  esac

  # 生成 config.json
  cat > config.json <<JSON
{
  "database": {
    "host": "${DB_HOST}",
    "port": ${DB_PORT},
    "user": "${DB_USER}",
    "password": "${DB_PASS}",
    "dbname": "${DB_NAME}",
    "sslmode": "${DB_SSL}"
  }
}
JSON
  ok "已生成 config.json"

  # 检查 go 环境
  command -v go >/dev/null 2>&1 || die "找不到 Go。请先安装 Go（>=1.26）：https://go.dev/dl/"
  ok "Go: $(go version)"

  # 要把前端内嵌进二进制，需要用 node 生成静态产物
  if command -v npm >/dev/null 2>&1; then
    info "构建前端静态产物…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && mkdir -p server/webui && cp -r web/out server/webui/dist
    info "编译内嵌前端的单一二进制…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "找不到 npm：只编译不内嵌前端的后端（前端用 npm run dev 单独启动）"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "编译完成 → ./artex"

  info "启动…（Ctrl-C 退出）"
  ./artex
}

echo "=============================="
echo "  ARTEX 安装"
echo "  1) 全部用 Docker 安装"
echo "  2) 本地运行（go 编译）"
echo "=============================="
case "$(ask '选择' 1)" in
  1) install_docker ;;
  2) install_local ;;
  *) die "选择无效" ;;
esac
