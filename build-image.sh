#!/usr/bin/env bash
# ARTEX 한국어판 — UI 언어를 골라 이미지를 빌드하고 기동한다.
#
# 왜 별도 스크립트가 필요한가
# --------------------------
# `Dockerfile` 은 "실행 전용"이라 **컨테이너 안에서 프런트엔드를 컴파일하지 않는다**.
# `COPY dist/<arch>/artex` 로 미리 만든 Linux 바이너리를 넣을 뿐이고, 그 바이너리 안에
# `server/webui/dist`(=`web/out`)가 embed 되어 있다. 즉 UI 언어는 **호스트에서
# `next build` 를 돌릴 때** 정해진다.
#
# 그래서 `NEXT_PUBLIC_LOCALE=zh docker compose up -d --build` 는 기대대로 동작하지
# 않는다. compose 는 Docker 레이어만 다시 만들 뿐이고, 그 레이어가 복사하는 바이너리는
# 그대로다(`COPY dist/...` 가 CACHED 로 끝난다). 이 스크립트는 그 앞단(프런트엔드 →
# 바이너리)을 대신 처리하고, 언어가 바뀌었을 때만 다시 빌드한다.
#
# 사용법
#   ./build-image.sh                      # .env 의 NEXT_PUBLIC_LOCALE(기본 ko)로 빌드 후 기동
#   ./build-image.sh --locale zh          # 중국어 UI 로 빌드 후 기동
#   ./build-image.sh --locale zh --no-up  # 이미지만 만들고 기동은 하지 않음
#   ./build-image.sh --force              # 언어가 같아도 강제로 다시 빌드
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*" >&2; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

# ── .env 를 읽어 설정을 한 곳에서 관리한다 ──────────────
load_env(){
  [ -f .env ] || return 0
  set +u                      # set -u 상태에서 .env 의 빈 값이 오류가 되지 않게 한다
  set -a; . ./.env; set +a
  set -u
}

LOCALE_OVERRIDE=""
DO_UP=1
FORCE=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --locale)
      [ "$#" -ge 2 ] || die "--locale 에는 값이 필요합니다(ko 또는 zh)"
      LOCALE_OVERRIDE="$2"; shift 2 ;;
    --no-up) DO_UP=0; shift ;;
    --force) FORCE=1; shift ;;
    --help|-h) sed -n '2,25p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "알 수 없는 인자입니다: $1(--help 참고)" ;;
  esac
done

load_env
# 우선순위: --locale 인자 > ARTEX_LOCALE(.env) > NEXT_PUBLIC_LOCALE > ko.
# ARTEX_LOCALE 을 먼저 보는 이유: 그 값이 컨테이너 런타임 언어(서버가 시드하는
# agent 표시 이름)까지 정하므로, UI 와 서버가 같은 값을 쓰도록 한 곳에서 온다.
LOCALE="${LOCALE_OVERRIDE:-${ARTEX_LOCALE:-${NEXT_PUBLIC_LOCALE:-ko}}}"
case "$LOCALE" in
  ko|zh) ;;
  *) die "지원하지 않는 locale 입니다: $LOCALE (ko 또는 zh)" ;;
esac

command -v go >/dev/null 2>&1 || die "Go 가 필요합니다(이미지에 넣을 바이너리를 컴파일합니다): https://go.dev/dl/"
command -v npm >/dev/null 2>&1 || die "Node.js/npm 이 필요합니다(프런트엔드 정적 빌드)"
command -v rsync >/dev/null 2>&1 || die "rsync 가 필요합니다"

# 컨테이너는 항상 Linux 이므로 호스트 OS 와 무관하게 GOOS=linux 로 컴파일한다.
ARCH="$(go env GOARCH)"
BIN="dist/${ARCH}/artex"
# 어느 언어로 이 바이너리를 만들었는지 기록한다. UI 언어는 바이너리에 embed 된 HTML 에
# 박혀 있어 밖에서 알아낼 방법이 없으므로, 빌드할 때 남겨 두고 다음 실행에서 비교한다.
MARKER="dist/${ARCH}/.locale"

need_build=0
reason=""
if [ "$FORCE" = "1" ]; then
  need_build=1; reason="--force"
elif [ ! -f "$BIN" ]; then
  need_build=1; reason="바이너리가 없습니다"
elif [ ! -f "$MARKER" ]; then
  need_build=1; reason="빌드 기록이 없습니다(이 스크립트로 만든 적이 없음)"
elif [ "$(cat "$MARKER")" != "$LOCALE" ]; then
  need_build=1; reason="마지막 빌드 언어는 $(cat "$MARKER") 입니다"
fi

if [ "$need_build" = "1" ]; then
  info "UI 언어 ${LOCALE} 로 다시 빌드합니다 (${reason})"
  info "① 프런트엔드 정적 빌드…"
  ( cd web && npm ci --include=dev && NEXT_EXPORT=1 NEXT_PUBLIC_LOCALE="$LOCALE" npx next build )
  info "② 내장 디렉터리로 동기화…"
  mkdir -p server/webui/dist
  rsync -a --delete web/out/ server/webui/dist/
  info "③ Linux/${ARCH} 바이너리 컴파일…"
  mkdir -p "dist/${ARCH}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build \
    -tags embedui -trimpath \
    -ldflags "-s -w -buildid= -X main.version=0.3.15-ko" \
    -o "$BIN" ./cmd/artex
  printf '%s' "$LOCALE" > "$MARKER"
  ok "바이너리 준비 완료: ${BIN} (UI 언어 ${LOCALE})"
else
  ok "이미 UI 언어 ${LOCALE} 로 빌드된 바이너리를 재사용합니다(다시 빌드하려면 --force)"
fi

if [ "$DO_UP" = "1" ]; then
  info "이미지를 빌드하고 기동합니다…"
  # 프런트엔드 locale 은 빌드 시점에 HTML 에 박히고, 서버는 그 값을 알 수 없다. 그래서
  # 서버가 DB 에 시드하는 표시 이름(reporter 등)의 언어는 런타임 값으로 넘겨 두 값을
  # 일치시킨다 — 이걸 빠뜨리면 중국어 UI 에 한국어 agent 이름이 섞인다.
  ARTEX_LOCALE="$LOCALE" docker compose up -d --build
  ok "완료 → http://localhost:8787 (UI 언어: ${LOCALE})"
  [ "$LOCALE" = "zh" ] && info "에이전트가 쓰는 리포트·요약·채팅 응답은 여전히 한국어입니다(Go 코드가 정함)."
  info "로그 확인: docker compose logs -f artex"
fi
