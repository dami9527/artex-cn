#!/usr/bin/env python3
"""정적 내보내기한 웹 UI(`web/out`)의 HTML 에 중국어(한자)가 새어 나왔는지 검사한다.

이 포크의 핵심 성과이자 차별점은 "사용자에게 보이는 화면에 중국어가 없다"는 것이다
(백로그 B4 가 수립한 `out` HTML 한자 0 속성). 그 속성을 손검증에만 맡기면, 기여자가
중국어 문자열을 하드코딩하거나 번역이 누락되는 회귀가 생겨도 다음 사람이 눈으로
발견할 때까지 조용히 깨진 채 남는다. 이 스크립트는 그 회귀를 머지 게이트에서 막는다.

검사 방식
---------
`web/out` 아래 모든 `.html` 파일의 **원시 내용**(렌더된 마크업뿐 아니라 인라인
스크립트·React Server Components 플라이이트 데이터까지)을 훑어 한자가 하나라도 있으면
종료 코드 1 로 끝난다. 원시 내용을 그대로 보는 이유는, 사용자에게 보이는 텍스트가
플라이이트 데이터(JSON 꼴 문자열)에 실려 HTML 에 함께 내려오기 때문이다. 태그만
벗겨 내면 그 경로의 누출을 놓친다.

검사 범위 밖 (정직한 한계)
-----------------------
이 게이트는 정적 내보내기 결과물만 본다. 그래서 **MOCK 모드(`NEXT_PUBLIC_MOCK=1`,
공개 데모가 쓰는 모드)에서 브라우저가 실행 중에 그려 내는 내용**까지는 검사하지 못한다.
`web/src/lib/mock/data.ts`·`handler.ts` 와 각 페이지의 하드코딩 샘플은 하이드레이션
뒤 `useEffect` 가 `mockHandle()` 로 가져와 React 상태로 렌더하므로, `out` HTML 이
아니라 런타임 JS 번들에만 실린다. 즉 이 경로로 들어온 중국어는 이 게이트를 통과한다
(과거 공개 데모의 `/system/logs` 샘플 로그에 중국어가 그렇게 조용히 남아 있었고,
백로그 G114 에서 한국어로 고쳤다).

그 표면은 소스 검토로 지킨다. 렌더 대상 파일(`web/src` 의 `.tsx`·클라이언트 데이터)에
남는 중국어는 반드시 다음 중 하나여야 한다: 코드 주석, `t.has(...)` 로 막혀 실제로는
쓰이지 않는 i18n 폴백, 백엔드 와이어 포맷 토큰(챗 멘션 분류 라벨·`[模型]` 결정 출처
센티넬), locale 가 `zh` 일 때만 보이는 분기 리터럴, 또는 성능 보존을 위해 번역하지
않는 에이전트 두뇌(段 [A]) 프롬프트를 그대로 비추는 미러. 이 가운데 어디에도 속하지
않고 "그냥 화면에 보이는 중국어"라면 한국어로 번역한다(백로그 G115 가 이 분류를
전수 확인했다).

허용·탐지 범위
-------------
- **한글(`가-힣` 등)은 허용**한다. 한자 블록과 한글 블록은 유니코드에서 겹치지 않으므로,
  한자 블록만 탐지하면 한글은 자동으로 통과한다.
- **한자(CJK 한자)만 탐지**한다: 통합 한자와 확장 A(`㐀`–`鿿`), 호환 한자
  (`豈`–`﫿`), 그리고 보충 평면의 확장 B 이상(`𠀀`–). 가나·전각 문장부호는
  "한자"가 아니므로 이 게이트의 대상이 아니다(원문 보존 코드 블록이나 증거 인용에
  섞여 들어와 거짓 양성을 내는 것을 막기 위해 범위를 한자로 좁힌다).

상류 보존이 필요한 의도적 한자
-----------------------------
현재 `out` HTML 에는 한자가 한 글자도 없어 허용 목록이 비어 있다. 앞으로 성능 보존을
위해 원문(중국어)을 일부러 화면에 남겨야 하는 경우(예: 언어 스위처의 `中文` 라벨)가
생기면, 그 글자를 아래 `ALLOWED_HAN` 에 추가하고 왜 남기는지 주석으로 적는다. 그래야
게이트는 엄격하게 유지되면서도 의도된 예외만 좁게 통과시킬 수 있다.

중국어 빌드(`NEXT_PUBLIC_LOCALE=zh`)
----------------------------------
locale 은 정적 내보내기 시점에 HTML 에 박히므로(web/src/i18n/config.ts 의
resolveLocale), `NEXT_PUBLIC_LOCALE=zh` 로 빌드하면 화면이 전부 중국어가 된다. 그
경우까지 이 게이트가 한자를 "누출"로 잡으면 의도한 중국어 빌드가 머지 게이트에서
막힌다. 그래서 `NEXT_PUBLIC_LOCALE` 이 지원 목록 안이고 그 값이 `zh` 이면 검사를
건너뛴다(종료 0). 값이 없거나 `ko` 이면 지금까지처럼 한자 0 을 강제한다 —
기본(한국어) 빌드에서 중국어가 새는 회귀는 그대로 잡는다.

표준 라이브러리만 쓰고 네트워크에 접속하지 않는다.
실행: `python3 -I scripts/check-web-cjk.py [out 디렉터리]`
(기본값은 저장소 루트의 `web/out`. CI 는 `web` 작업 디렉터리에서
`python3 -I ../scripts/check-web-cjk.py` 로 부른다.)
"""
import json
import os
import re
import subprocess
import sys

# web/src/i18n/config.ts 의 LOCALES·DEFAULT_LOCALE 과 같은 값. 표준 라이브러리만 쓰는
# 스크립트라 TypeScript 를 import 할 수 없어 복제하고, verify_locale_source() 로
# 원본과 어긋나지 않았는지 확인한다(조용히 갈라지면 게이트가 엉뚱한 기준으로 돈다).
LOCALES = ("ko", "zh")
DEFAULT_LOCALE = "ko"

# CJK 한자 블록만. 한글(AC00–D7A3)·가나·전각 문장부호는 들어 있지 않다.
# 범위: 확장 A + 통합 한자(U+3400–U+9FFF = 㐀–鿿), 호환 한자(U+F900–U+FAFF = 豈–﫿),
#       보충 평면 확장 B 이상(U+20000–U+2FFFF).
HAN = re.compile(r"[\u3400-\u9fff\uf900-\ufaff\U00020000-\U0002ffff]")

# 화면에 일부러 남기는 한자(예: 언어 스위처 `中文`). 지금은 비어 있다 —
# 비우면 어떤 한자든 게이트에 걸린다. 예외를 더할 때는 글자와 사유를 함께 적는다.
# 예) ALLOWED_HAN = {"中", "文"}  # 언어 스위처 '中文' 라벨(상류 대조용)
ALLOWED_HAN: set = set()

# zh.json 에서 영어로 두는 것이 맞는 값(= 중국어 UI 에서도 영어를 쓰는 기술 용어).
# 한국어판은 이런 단어를 한글로 옮겼지만(에이전트·스킬), 중국어 UI 에서는 영문 표기가
# 관용이다. 새 항목을 추가할 때는 왜 영문이 맞는지 이 목록 옆에 적는다.
# 이 목록에 없는 값이 zh 에서 영어로 나타나면 아래 카탈로그 검사가 회귀로 잡는다.
ALLOWED_ZH_LATIN: set = {
    "Agent",    # 화면·문서 모두 「Agent」로 통용(에이전트보다 우세)
    "Skill",    # ARTEX 의 기능 단위 이름. 상류 원문도 영문 유지
    "Search",   # 상단 검색 버튼
    "App",      # 자산 유형 라벨(엔드포인트/服务 등과 나란히 쓰임)
    "cache",    # token 사용량 4종 라벨(input/cache/output) 중 하나
    "input",
    "output",
}



def resolve_locale() -> str:
    """빌드에 쓰인 locale 을 정한다(web/src/i18n/config.ts 의 resolveLocale 과 같은 규칙)."""
    raw = os.environ.get("NEXT_PUBLIC_LOCALE")
    return raw if raw in LOCALES else DEFAULT_LOCALE


def verify_locale_source() -> None:
    """config.ts 의 LOCALES·DEFAULT_LOCALE 과 위 복제값이 일치하는지 확인한다."""
    try:
        root = subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"], text=True
        ).strip()
    except (subprocess.CalledProcessError, FileNotFoundError):
        return  # git 이 없으면 대조만 건너뛴다(검사 자체는 계속한다)
    try:
        with open(
            os.path.join(root, "web", "src", "i18n", "config.ts"), encoding="utf-8"
        ) as fh:
            src = fh.read()
    except OSError:
        return
    m = re.search(r"LOCALES\s*=\s*\[([^\]]*)\]", src)
    if m:
        found = tuple(re.findall(r'"([^"]+)"', m.group(1)))
        if found and found != LOCALES:
            raise SystemExit(
                f"오류: web/src/i18n/config.ts 의 LOCALES={found} 가 이 스크립트의 "
                f"LOCALES={LOCALES} 와 다릅니다. 둘을 맞추세요."
            )
    m = re.search(r'DEFAULT_LOCALE[^=]*=\s*"([^"]+)"', src)
    if m and m.group(1) != DEFAULT_LOCALE:
        raise SystemExit(
            f"오류: web/src/i18n/config.ts 의 DEFAULT_LOCALE={m.group(1)!r} 가 "
            f"이 스크립트의 {DEFAULT_LOCALE!r} 와 다릅니다. 둘을 맞추세요."
        )


def check_catalog_fallbacks(root: str) -> int:
    """zh.json 의 값이 아직 영어인데 ko.json 은 번역돼 있으면 알린다.

    이 게이트가 잡는 다른 유형의 누출: 화면에 나가는 값이 `zh.json` 안에서 **영어로**
    남아 있는 경우다. 통합 한자 검사로는 절대 걸리지 않는데(영어니까), 실제로 그렇게
    새어 나갔다 — `notFound` 세 키와 `search.empty` 가 `zh.json` 에서 영어였고, 그래서
    중국어 UI 빌드의 404 가 Next 기본 영어 페이지로 보였다. ko.json 을 기준선으로 삼아
    "ko 는 한국어인데 zh 는 영어" 인 값만 보고하므로, 양쪽 모두에서 관용적으로 영어를
    쓰는 기술 용어(Bot Token 등)는 걸리지 않는다.
    """
    try:
        with open(os.path.join(root, "web", "messages", "zh.json"), encoding="utf-8") as fh:
            zh = json.load(fh)
        with open(os.path.join(root, "web", "messages", "ko.json"), encoding="utf-8") as fh:
            ko = json.load(fh)
    except (OSError, ValueError) as exc:
        print(f"경고: 메시지 카탈로그를 읽지 못해 검사를 건너뜁니다 — {exc}")
        return 0

    def flat(node: object, prefix: str = "") -> dict:
        out = {}
        if isinstance(node, dict):
            for k, v in node.items():
                out.update(flat(v, f"{prefix}.{k}"))
        else:
            out[f"{prefix}"] = node
        return out

    def is_latin(text: str) -> bool:
        return bool(text) and all(ord(c) < 128 for c in text)

    def has_non_latin(text: str) -> bool:
        return not is_latin(text)

    offenders = []
    ko_flat, zh_flat = flat(ko), flat(zh)
    for key, value in sorted(zh_flat.items()):
        other = ko_flat.get(key)
        if not isinstance(value, str) or not isinstance(other, str):
            continue
        # zh 가 영어이고 ko 는 한국어/한자 → zh 쪽이 미번역일 가능성이 높다.
        # 단, 중국어 UI 에서도 영문을 쓰는 기술 용어는 허용한다.
        if is_latin(value) and has_non_latin(other) and value not in ALLOWED_ZH_LATIN:
            offenders.append((key, value, other))

    if offenders:
        print(f"중국어 카탈로그 미번역 의심 {len(offenders)}건 — zh.json 값이 영어입니다:")
        for key, value, other in offenders:
            print(f"  {key}\n    zh: {value[:70]!r}\n    ko: {other[:70]!r}")
        print(
            "\nzh.json 의 해당 값을 중국어로 채우거나, 상류 원문 그대로 두는 것이 맞다면 "
            "이 검사의 기준선(ko.json)을 확인하세요."
        )
        return 1
    print("중국어 카탈로그 미번역 의심 0 — zh.json 의 값이 모두 번역돼 있습니다.")
    return 0


def find_out_dir() -> str:
    """검사할 `out` 디렉터리를 정한다.

    인자로 경로를 주면 그것을, 없으면 git 저장소 루트의 `web/out` 를 쓴다.
    """
    if len(sys.argv) > 1:
        return os.path.abspath(sys.argv[1])
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    return os.path.join(root, "web", "out")


def main() -> int:
    verify_locale_source()

    # 카탈로그 검사는 빌드 산출물이 아니라 소스(web/messages)를 보므로 locale 과 무관하게
    # 먼저 돈다. zh 빌드에서도 영어로 남은 값은 그대로 문제이기 때문이다.
    try:
        root = subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"], text=True
        ).strip()
    except (subprocess.CalledProcessError, FileNotFoundError):
        root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    catalog_rc = check_catalog_fallbacks(root)

    if resolve_locale() == "zh":
        print(
            "NEXT_PUBLIC_LOCALE=zh — 중국어 UI 빌드이므로 한자 검사를 건너뜁니다"
            "(의도된 중국어이며 누출이 아님)."
        )
        return catalog_rc

    out_dir = find_out_dir()
    if not os.path.isdir(out_dir):
        print(
            f"오류: 검사할 디렉터리가 없습니다 — {out_dir}\n"
            "먼저 `npm run build:static` 로 정적 내보내기를 만든 뒤 실행하세요."
        )
        return 2

    checked = 0
    offenders = []  # (상대경로, 한자 개수, 미리보기)
    for dirpath, _dirnames, filenames in os.walk(out_dir):
        for name in filenames:
            if not name.endswith(".html"):
                continue
            checked += 1
            path = os.path.join(dirpath, name)
            with open(path, encoding="utf-8", errors="ignore") as fh:
                text = fh.read()
            found = [ch for ch in HAN.findall(text) if ch not in ALLOWED_HAN]
            if found:
                rel = os.path.relpath(path, out_dir)
                kinds = "".join(sorted(set(found)))
                offenders.append((rel, len(found), kinds))

    print(f"검사한 HTML {checked}개 (대상: {out_dir})")
    if offenders:
        print(f"중국어(한자) 누출 {len(offenders)}개 파일 — 사용자 노출 HTML 에 한자가 있습니다:")
        for rel, count, kinds in offenders:
            preview = kinds if len(kinds) <= 30 else kinds[:30] + "…"
            print(f"  out/{rel}: 한자 {count}개 · 종류 {preview}")
        print(
            "\n한글로 번역하거나, 상류 보존이 꼭 필요한 의도적 한자라면 "
            "scripts/check-web-cjk.py 의 ALLOWED_HAN 에 사유와 함께 추가하세요.\n"
            "중국어 UI 로 일부러 빌드한 것이라면 `NEXT_PUBLIC_LOCALE=zh` 를 설정한 뒤 "
            "다시 실행하세요(그 경우 이 검사를 건너뜁니다)."
        )
        return 1
    if checked == 0:
        print("경고: 검사한 HTML 이 없습니다 — 빌드가 제대로 되었는지 확인하세요.")
        return 2
    print("중국어(한자) 누출 0 — 사용자 노출 HTML 이 전부 한국어/비한자입니다.")
    return catalog_rc


if __name__ == "__main__":
    sys.exit(main())
