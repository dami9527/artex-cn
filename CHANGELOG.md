# 변경 이력

한국어 · [English](CHANGELOG.en.md) · [中文(원본·상류)](CHANGELOG.zh.md)

이 문서는 ARTEX 한국어판(이 포크)이 상류 저장소에 더한 변경을 기록합니다. 형식은 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) 를 참고합니다.

상류 ARTEX 프로젝트의 버전별 릴리스 이력(0.3.x 이하)과 기여자 목록은 원본 중국어 그대로 [`CHANGELOG.zh.md`](CHANGELOG.zh.md) 에 보존했습니다. 상류 변경과 대조하기 쉽도록 `README.zh.md` 와 같은 방식으로 원문을 그대로 남깁니다. 각 변경의 자세한 내용과 근거는 저장소 커밋 이력에서 확인할 수 있습니다.

## [Unreleased] · 한국어판 변경

### Docker 배포 경로 복구

- **Docker 로 한국어판을 띄울 수 있게 했습니다.** `docker-compose.yml` 의 `artex` 서비스는 `autumn27/artex` 이미지를 받도록 되어 있었는데, 이 이미지는 원작자가 저장소를 닫으면서 Docker Hub 에서 사라졌습니다(현재 pull 하면 `not found`, `autumn27` 네임스페이스에는 `scopesentry` 두 개만 남아 있습니다). 그래서 compose 에 `build:` 를 넣고 이미지 이름을 로컬 태그(`${ARTEX_IMAGE:-artex-ko:local}`)로 바꿔, `docker compose up -d --build` 한 줄로 한국어판 이미지를 직접 빌드해 기동하도록 했습니다. 상류 이미지를 쓰던 경로(`docker compose pull`)는 더 이상 성립하지 않습니다.
- **`Dockerfile` 이 "실행 전용"이라는 전제를 문서와 스크립트에 반영했습니다.** 이 Dockerfile 은 컨테이너 안에서 컴파일하지 않고 `COPY dist/<arch>/artex` 로 미리 만든 Linux 바이너리를 넣습니다. 따라서 ① 프런트엔드 정적 빌드 → ② `GOOS=linux` 크로스 컴파일 → ③ `docker build` 순서가 필요하고, 앞 단계를 건너뛰면 `COPY` 에서 실패합니다. README 에 ["한국어판 이미지를 직접 빌드하기"](README.md#한국어판-이미지를-직접-빌드하기) 절을 새로 두어 이 순서와 `--platform`·`GOARCH` 일치 주의점, 그리고 `build.sh` 와 Dockerfile 이 기대하는 경로가 다르다는 점(`dist/artex-linux-amd64/` vs `dist/<arch>/`)을 적었습니다.
- **`install.sh` 의 "① 전부 Docker" 와 `update.sh` 의 Docker 경로가 사라진 이미지를 받던 문제를 고쳤습니다.** 이제 두 스크립트 모두 현재 소스로 이미지를 다시 빌드합니다(`docker compose up -d --build`). 필요한 도구(Go·Node.js/npm·rsync)가 없으면 그 자리에서 안내하고 멈춥니다. `update.sh` 에서 더 이상 쓰이지 않는 `ARTEX_TAG` 입력 단계는 제거했습니다.
- **`.env.example` 을 실제 동작에 맞췄습니다.** `ARTEX_TAG`(상류 이미지 태그) 대신 `ARTEX_IMAGE`(로컬로 빌드한 이미지 태그)를 두고, 이 저장소가 이미지를 배포하지 않는다는 점을 명시했습니다.
- **비밀번호를 나중에 바꿔 생기는 기동 실패를 문서화했습니다.** PostgreSQL 공식 이미지는 `POSTGRES_PASSWORD` 를 **볼륨이 비어 있을 때 한 번만** 읽어 `initdb` 에 쓰므로, 스택을 이미 한 번 올린 뒤 `.env` 의 비밀번호를 바꾸면 그 값이 무시되고 artex 만 새 비밀번호로 접속해 `28P01` 인증 실패 → `비정상 종료 (code=1)` 재시작 루프에 빠집니다(포트는 Docker 가 잡고 있어도 컨테이너 안에 리스닝 프로세스가 없어 브라우저는 연결 실패로 보입니다). README(ko·en)에 ["페이지가 안 열릴 때 (Docker)"](README.md#페이지가-안-열릴-때-docker) 절을 두어 로그 확인 방법과 두 가지 복구법(`down -v` 후 재초기화 / `ALTER USER` 로 볼륨 데이터를 지키며 비밀번호 정렬)을 적고, `.env.example` 에도 같은 경고를 넣었습니다. 순서는 항상 **`.env` 먼저 → `docker compose up -d --build`** 입니다.

### UI 언어 선택 (한국어 / 중국어)

- **`build-image.sh` 를 추가해 UI 언어 전환을 한 명령으로 만들었습니다.** 앞선 문서·스크립트는 "빌드 경로가 `NEXT_PUBLIC_LOCALE` 을 넘기게" 하는 데까지만 손댔는데, **`docker compose up -d --build` 앞에 이 변수를 붙여도 언어가 바뀌지 않는다**는 사실을 놓쳤습니다. 이 프로젝트의 `Dockerfile` 은 컨테이너 안에서 프런트엔드를 컴파일하지 않고(`COPY dist/<arch>/artex` 로 미리 만든 바이너리를 넣음) 그 바이너리에 `web/out` 이 embed 되어 있어서, compose 는 그 복사를 `CACHED` 로 끝낼 뿐입니다. 즉 UI 언어는 **호스트에서 `next build` 를 돌릴 때** 정해집니다. `build-image.sh` 가 ① 프런트엔드 → ② 내장 디렉터리 동기화 → ③ `GOOS=linux` 컴파일 → ④ `docker compose up -d --build` 를 대신 하고, `dist/<arch>/.locale` 에 마지막 빌드 언어를 기록해 **언어가 바뀌었을 때만 다시 빌드**합니다(같으면 재사용, `--force` 로 강제, `--no-up` 으로 기동 생략). 사용법은 `./build-image.sh --locale zh` 처럼 씁니다.
- **`install.sh`·`update.sh` 의 빌드 로직을 `build-image.sh` 로 위임했습니다.** 같은 로직을 두 벌 두면 UI 언어가 조용히 어긋나므로(실제로 그렇게 어긋났습니다) 한 곳에 모으고, 두 스크립트는 호출만 합니다.
- **`NEXT_PUBLIC_LOCALE` 로 UI 언어를 고를 수 있게 했습니다(기본 `ko` = 한국어판).** `web/src/i18n/config.ts` 에는 이미 `LOCALES = ["ko","zh"]`·`DEFAULT_LOCALE = "ko"`·`resolveLocale()` 이 있었지만, 정작 **빌드 경로 어디에서도 `NEXT_PUBLIC_LOCALE` 을 넘기지 않아** 중국어 UI 를 만들 방법이 없었습니다(항상 기본값으로 떨어짐). locale 은 정적 내보내기 시점에 HTML 에 박히므로 런타임 전환이 아니라 빌드 시점 선택입니다.
- **`scripts/check-web-cjk.py` 를 locale 인지로 바꿨습니다.** 이 게이트는 "사용자 노출 HTML 에 한자 0" 을 강제하는데, 의도한 중국어 빌드까지 한자 누출로 잡으면 머지 게이트가 막힙니다. 이제 `NEXT_PUBLIC_LOCALE=zh` 가 설정돼 있으면 검사를 건너뛰고(종료 0), 값이 없거나 `ko` 이면 지금까지처럼 한자 0 을 강제합니다. 실패 메시지에도 "의도한 중국어 빌드라면 `NEXT_PUBLIC_LOCALE=zh` 를 설정하라" 는 안내를 넣었습니다. 스크립트가 `LOCALES`·`DEFAULT_LOCALE` 을 복제해 쓰므로, `config.ts` 의 값과 어긋나면 그 자리에서 오류로 알립니다(게이트가 엉뚱한 기준으로 도는 것을 막습니다).
- **바뀌는 것은 화면 문구뿐임을 문서에 명시했습니다.** 에이전트가 쓰는 취약점 리포트·사실 요약·최종 요약·채팅 응답의 언어는 Go 코드(`agent/prompt.go` 의 `langDirective()`)가 한국어로 고정하며 이 값과 무관합니다. 그 출력까지 중국어로 바꾸려면 `langDirective()` 와 그 계약을 검증하는 `agent/prompt_test.go` 를 함께 고쳐야 하고, 이는 이 포크의 한국어화 설계를 되돌리는 변경입니다.

### 상류 동기화 (v0.3.15)

포크 지점(상류 `d003372`, 2026-10-03) 이후 상류가 올린 커밋 세 건을 이 저장소로 가져왔습니다. 셋 다 포크 이후 시점의 상류 변경이라 병합 충돌은 한국어화가 이미 손댄 파일에만 국한됐고, 그 지점은 아래처럼 한국어판 규약에 맞춰 해소했습니다.

- **모델 폴백 승인의 token 사용량을 따로 계량합니다([`db/llm_usage.go`](db/llm_usage.go) · `server/intercept.go`).** 승인 호출에는 지금까지 독립된 계량 귀속이 없어 얼마를 썼는지 알 수 없었습니다. 이제 `worker=judge` 로 따로 집계하고, 「시스템 → 가로채기」의 「모델 폴백 승인」 스위치 아래에 사용량 카드(호출 횟수·입력·출력·캐시 읽기·쓰기, 최근 30일 일별 막대)를 둡니다. 상류는 이 카드의 문구를 중국어로 하드코딩했지만, 이 저장소는 `web/messages/ko.json`·`zh.json` 에 `interceptPage.judgeUsage.*` 키를 새로 두어 기존 화면과 같은 i18n 경로로 옮겼습니다(`zh.json` 에는 대조용 중국어 원문을 함께 넣었습니다).
- **인증 초기화의 fail-open 을 막았습니다(상류 `a951e4a`, `server/auth.go` · `db/settings.go`).** 비밀번호 관련 읽기가 실패했을 때 이를 「설정되지 않음」으로 취급하면 인증되지 않은 요청이 이미 설정된 관리자 비밀번호를 덮어쓸 수 있었습니다. 이제 읽기 실패는 503 으로 돌려주고, 최초 1회만 설정되도록 upsert 대신 기본 키 제약(`INSERT ... ON CONFLICT DO NOTHING`)에 보장을 두며, 비밀번호 길이 검증(8자 이상, bcrypt 상한 72바이트)을 서버에서 강제합니다. 한국어판은 새 오류 문구와 검증 메시지를 한국어로 옮기고, setup 화면의 「확인 불가」 상태 문구를 `auth.setup.*` 키로 추가했습니다.
- **상류 `v0.3.15` 의 변경 이력을 [`CHANGELOG.zh.md`](CHANGELOG.zh.md) 에 반영했습니다.** 상류의 정본 커밋(定版)은 이 저장소의 분리된 변경 이력 구조와 맞지 않아 그대로 cherry-pick 하지 않고, `[Unreleased]` 에 있던 내용을 `## [0.3.15] - 2026-10-07` 로 정리하는 방식으로 원문을 옮겼습니다.

### 현지화 (i18n)

- **사용자 노출 출력을 한국어로 강제했습니다.** 벤치마크된 에이전트의 행동 지침 본문(두뇌)은 성능 보존을 위해 원문 그대로 두고, 코드 고정 세그먼트(`langDirective`)로 사용자에게 보이는 산출물(취약점 리포트, 사실 요약, 최종 요약, 채팅 응답)만 한국어로 작성하도록 지시합니다. 명령·페이로드·코드·로그 원문은 원본을 보존합니다.
- **웹 UI 를 한국어로 옮겼습니다.** Next App Router 에 `next-intl` 을 도입하고 문자열을 `web/messages/ko.json` 과 `web/messages/zh.json` 으로 분리했습니다. 원본 중국어는 `zh.json` 에 보존해 상류 업데이트와 대조합니다. 대시보드·취약점·대화·알림 발송·가로채기·LLM 설정 등 화면 문자열을 한국어로 옮겼습니다.
- **서버 API 의 사용자 노출 오류·응답을 한국어로 옮겼습니다.** 브라우저로 돌아가는 HTTP 오류·응답 문구를 한국어로 교체했습니다. 단 에이전트 두뇌의 입력으로 되먹여지는 문구는 벤치마크 드리프트를 막기 위해 원문을 유지했고, 그 판정 근거는 저장소 작업 문서에 기록했습니다.
- **문서를 한국어로 정비했습니다.** 한국어 `README.md` 를 만들고 영어 `README.en.md` 를 함께 두었으며, 원본 중국어는 `README.zh.md` 로 보존했습니다.

### 방어·탐지 자료

- **방어·탐지 가이드를 추가했습니다.** 자율 AI 공격이 기존 스캐너와 무엇이 다른가, 방어자가 관측할 수 있는 지문(IoC), 진입점과 하드닝, 탐지 규칙, 사고 대응을 정리한 한국어 가이드([`docs/defense-ko.md`](docs/defense-ko.md))와 같은 내용의 영어판([`docs/defense-en.md`](docs/defense-en.md))을 두었습니다.
- **배포용 탐지 규칙을 제공합니다.** 가이드의 지문을 바로 쓸 수 있는 규칙으로 옮겼습니다. 호스트·로그 계층은 [Sigma](https://sigmahq.io) 원자·상관 규칙([`detections/sigma/`](detections/sigma/)), 네트워크 계층은 enrich 프로브와 norma SDK WebFetch 의 User-Agent 를 겨냥한 [Suricata](https://suricata.io) 규칙([`detections/suricata/`](detections/suricata/))으로 담았습니다.
- **ATT&CK 커버리지를 가시화했습니다.** 규칙이 태깅하는 기법을 MITRE ATT&CK Navigator 레이어([`detections/attack/`](detections/attack/))로 정리했습니다.
- **기계 판독 침해지표(IoC)를 표준 형식으로 제공합니다.** ARTEX 가 내보내는 고유 지문을 한 파일로 모은 CSV([`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv))와, 같은 지표를 위협 인텔리전스 플랫폼에 바로 가져올 수 있는 MISP 이벤트([`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json))로 담았습니다. 규칙이 받쳐 주는 지표는 `to_ids` 로, 호스트 포렌식 포트는 분류용 단서로 구분해 표기합니다.
- **재현 가능한 탐지 테스트를 붙였습니다.** 규칙을 실제로 돌려 증명하는 테스트 여덟 종(Sigma 구조·컴파일 검증, Sigma 실시간 이벤트 매칭, 백엔드 이식성, SigmaHQ 관례 린트, Suricata 로드·발화, ATT&CK 레이어 정합, 지표-소스 일치, MISP 내보내기 ↔ CSV 동기화)과 이를 한 번에 돌리는 일괄 러너·pre-commit 예시를 추가하고 CI 머지 게이트로 연결했습니다. Sigma 실시간 이벤트 매칭은 규칙이 컴파일될 뿐 아니라 악성 샘플 이벤트에는 실제로 발화하고 정상 이벤트에는 침묵하는지까지 원자·상관 규칙 모두에서 확인합니다.

### 저장소 정비

- **보안·오남용 경고와 국내법 고지를 넣었습니다.** README 최상단에 사용 범위, 정보통신망법·개인정보보호법 고지, 오남용 금지 경고를 추가했습니다.
- **한국어 UI 스크린샷으로 화면 미리 보기를 교체했습니다.**
- **메인테이너 런북과 기여 가이드를 정비했습니다.** 상류 동기화·번역 드리프트를 막기 위한 런북([`MAINTAINING.md`](MAINTAINING.md))과 탐지 규칙 기여 계약([`CONTRIBUTING.md`](CONTRIBUTING.md))을 두었습니다. 런북에는 릴리스 발행 파이프라인의 빌드 전제와, 태그 없이 로컬에서 그 전제를 검증하는 절차도 함께 정리했습니다.
- **푸시·PR 머지 게이트 CI 를 추가했습니다.** 상류 저장소는 태그 릴리스에서만 CI 가 돌았지만, 이 포크는 모든 푸시와 PR 에서 Go 빌드·정적 분석(`go vet`)·단위 테스트([`ci.yml`](.github/workflows/ci.yml)), 한국어 UI 정적 빌드([`web.yml`](.github/workflows/web.yml)), 문서의 저장소 내부 링크·이미지 참조 무결성([`docs.yml`](.github/workflows/docs.yml))을 돌려, 한국어화 과정에서 생긴 회귀를 머지 전에 잡습니다. 데이터베이스가 있어야 하는 통합 테스트는 패키지마다 격리된 PostgreSQL 서비스로 함께 검증합니다. 문서 링크 검사는 외부 네트워크에 의존하지 않는 결정론적 스크립트([`scripts/check-doc-links.py`](scripts/check-doc-links.py))로 돌려, 다국어 문서가 서로를 가리키는 많은 상대 링크와 화면 미리 보기 이미지가 깨진 채 머지되는 것을 막습니다. 문서 앵커(`#헤딩`) 링크도 GitHub 과 같은 slug 규칙으로 헤딩과 대조해, 헤딩 글자가 바뀌어 조용히 끊긴 목차·상호 참조 링크를 함께 잡습니다. 탐지 규칙 스위트는 위 '방어·탐지 자료' 절에서 설명한 머지 게이트가 담당합니다.
- **외부 링크 생존을 주기적으로 점검합니다.** 방어 가이드가 가리키는 사고 신고 창구·표준 참조 같은 외부 링크는 원격 서버 상태에 의존해 flaky 하므로 머지 게이트에서 빼고, 비차단 워크플로([`external-links`](.github/workflows/external-links.yml))가 매주 월요일과 수동 실행으로 브라우저 User-Agent·GET·리다이렉트 추적 점검([`scripts/check-external-links.py`](scripts/check-external-links.py))을 돌립니다. 호스트는 살아 있는데 확인 방법만 막힌 경우(봇 차단·속도 제한)와 우리가 고칠 수 없는 상류 상속 죽은 링크(allowlist)는 실패로 치지 않아, 우리 문서가 큐레이션한 외부 링크가 새로 깨질 때만 빨갛게 드러냅니다.
- **기여·거버넌스 인프라를 갖췄습니다.** 버그·기능·번역 이슈 템플릿([`.github/ISSUE_TEMPLATE/`](.github/ISSUE_TEMPLATE/))과 풀 리퀘스트 템플릿([`PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md)), 보안 취약점 신고 정책([`SECURITY.md`](SECURITY.md)), 행동 강령([`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md))을 두어, 외부 기여자가 이슈·PR·보안 신고를 일관된 양식으로 제출하도록 했습니다.
- **해외 기여자를 위한 영어 문서 레이어를 완성했습니다.** 이 저장소는 한국어가 주 언어이지만, 한국어를 읽지 못하는 기여자·보안 연구자·방어자가 같은 정보에 도달하도록 핵심 문서의 영어판을 함께 두었습니다. 영어 `README.en.md`·방어 가이드([`docs/defense-en.md`](docs/defense-en.md))에 더해, 변경 이력([`CHANGELOG.en.md`](CHANGELOG.en.md)), 보안 신고 정책([`SECURITY.en.md`](SECURITY.en.md)), 행동 강령([`CODE_OF_CONDUCT.en.md`](CODE_OF_CONDUCT.en.md)), 기여 가이드([`CONTRIBUTING.en.md`](CONTRIBUTING.en.md)), 메인테이너 런북([`MAINTAINING.en.md`](MAINTAINING.en.md)), 트래픽 증거 설계 문서([`docs/finding-traffic-evidence-en.md`](docs/finding-traffic-evidence-en.md)), 그리고 버그·기능·번역 이슈 템플릿의 영어판을 갖췄습니다. 한국어판과 영어판은 머리말에서 서로를 가리켜, 어느 언어로 들어와도 반대쪽으로 이동할 수 있습니다. (풀 리퀘스트 템플릿은 현재 한국어판만 제공합니다.)

---

상류 ARTEX 프로젝트의 버전별 릴리스 이력과 기여자 목록은 [`CHANGELOG.zh.md`](CHANGELOG.zh.md) 에서 원문 그대로 볼 수 있습니다.
