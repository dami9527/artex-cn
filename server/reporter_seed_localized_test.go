package server

import (
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/config"
)

// hasHan 은 문자열에 중국어 한자가 있는지 본다(중국어 라벨 판정용). 서버 패키지에는
// 이런 헬퍼가 없어 여기서 둔다 — assertKoreanError 는 반대 방향(한자 0)만 본다.
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// hasHangul 은 문자열에 한글이 있는지 본다. 중국어 라벨에 한국어가 섞여 나가는 회귀를
// 잡기 위한 것으로, 한국어 라벨 쪽 판정은 assertKoreanError 가 맡는다.
func hasHangul(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

// TestReporterSeedLabelsLocalized 는 시드되는 reporter 에이전트의 표시 전용 라벨
// (reporterAgentName·reporterAgentDescription)이 **런타임 locale 에 맞는 언어**로
// 유지되는지 지키는 회귀 방어 테스트다. 이 두 문자열은 `agentDTO`(server_mgmt.go)로
// system/agents UI 에만 렌더되고, 어떤 에이전트 system 프롬프트에도(renderSystem 은
// key→段[A] agent.ReporterDefaultPrompt 로만 조립) 모델 기반 에이전트 선택에도
// (reporter 트리거는 report_finding 도구 호출 기반 OnToolCall·결정적) 들어가지 않는
// 순수 UI 라벨이다. 그래서 번역해도 BRIEF 경계 #1(두뇌 미번역)을 건드리지 않는다.
// 반대로 reporter 의 두뇌 본문(agent.ReporterDefaultPrompt)과 트리거 주입
// 메시지(reporterToolCallMessage)는 모델 입력이라 중국어 원문을 보존하므로 이 테스트
// 대상이 아니다. [[F35]] [[G133]]
//
// 원래는 "항상 한국어"를 강제했지만, 중국어 UI 로 빌드한 배포에서는 화면에 한국어
// 이름이 그대로 나갔다(사용자 제보). 그래서 locale 에 따라 한국어/중국어를 고르도록
// 바뀌었고, 이 테스트는 **두 locale 각각에서 기대한 언어가 나오는지**를 지킨다 —
// 한쪽 값만 고치고 다른 쪽을 빠뜨리는 회귀를 잡는다.
func TestReporterSeedLabelsLocalized(t *testing.T) {
	cases := []struct {
		locale string
		name   string
		desc   string
		han    bool // 중국어(한자 포함) 라벨이어야 하는가
	}{
		{"ko", reporterAgentName, reporterAgentDescription, false},
		{"zh", reporterAgentNameZh, reporterAgentDescriptionZh, true},
	}
	for _, c := range cases {
		t.Setenv("ARTEX_LOCALE", c.locale)
		if got := config.Locale(); got != c.locale {
			t.Fatalf("ARTEX_LOCALE=%s 인데 config.Locale()=%s", c.locale, got)
		}
		name, desc := reporterAgentLabels()
		if name != c.name || desc != c.desc {
			t.Fatalf("locale %s: reporterAgentLabels() = (%q, %q), 기대 (%q, %q)", c.locale, name, desc, c.name, c.desc)
		}
		if c.han {
			// 한자 포함(중국어)이어야 하고, 한국어 라벨이 섞여 나가면 실패한다.
			if !hasHan(name) || !hasHan(desc) {
				t.Fatalf("locale %s: 중국어 라벨에 한자가 없습니다 (name=%q desc=%q)", c.locale, name, desc)
			}
			if hasHangul(name) || hasHangul(desc) {
				t.Fatalf("locale %s: 중국어 라벨에 한글이 섞였습니다 (name=%q desc=%q)", c.locale, name, desc)
			}
		} else {
			assertKoreanError(t, "reporter_name", name)
			assertKoreanError(t, "reporter_description", desc)
		}
	}
}

// TestRetesterSeedLabelsLocalized 도 같은 계약을 retester(취약점 재검증 에이전트)에
// 적용한다. 이 이름·설명도 시스템 화면에만 렌더되는 순수 UI 라벨이다(도구 설명·
// 파라미터 설명·actool.Errorf 같은 두뇌 입력은 별개로 원문을 보존한다 —
// finding_retests.go 상단 주석 참조).
func TestRetesterSeedLabelsLocalized(t *testing.T) {
	cases := []struct {
		locale string
		name   string
		desc   string
		han    bool
	}{
		{"ko", retesterAgentName, retesterAgentDescription, false},
		{"zh", retesterAgentNameZh, retesterAgentDescriptionZh, true},
	}
	for _, c := range cases {
		t.Setenv("ARTEX_LOCALE", c.locale)
		name, desc := retesterAgentLabels()
		if name != c.name || desc != c.desc {
			t.Fatalf("locale %s: retesterAgentLabels() = (%q, %q), 기대 (%q, %q)", c.locale, name, desc, c.name, c.desc)
		}
		if c.han {
			if !hasHan(name) || !hasHan(desc) {
				t.Fatalf("locale %s: 중국어 라벨에 한자가 없습니다 (name=%q desc=%q)", c.locale, name, desc)
			}
			if hasHangul(name) || hasHangul(desc) {
				t.Fatalf("locale %s: 중국어 라벨에 한글이 섞였습니다 (name=%q desc=%q)", c.locale, name, desc)
			}
		} else {
			assertKoreanError(t, "retester_name", name)
			assertKoreanError(t, "retester_description", desc)
		}
	}
}
