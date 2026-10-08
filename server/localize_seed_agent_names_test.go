package server

import (
	"testing"

	"github.com/Autumn-27/artex/config"
)

// TestLocalizeSeedAgentNames 는 localizeSeedAgentNames 의 계약을 실제 DB 로 검증한다.
//
// 이 테스트가 필요한 이유: 시드 라벨은 문자열로 DB 에 들어가고 시드는 1회만 돌아,
// ① 언어를 바꿔도 이름이 남고 ② 매 기동 덮어쓰면 사용자가 UI 에서 고친 이름을 잃는다.
// 그 사이를 잡는 것이 localizeSeedAgentNames 인데, 초기 구현에 버그가 있었다 —
// "이미 목표값이면 건너뛴다"를 **목표값**과 비교해서, 목표가 우연히 다른 항목(legacy
// 중국어)의 기본값과 같은 경우 뒤 항목이 통째로 건너뛰어져 한국어 전환이 안 됐다.
// 정적 테스트(reporter_seed_localized_test.go)는 상수만 보므로 이 버그를 잡지 못한다 —
// DB 상태 전이를 보는 이 테스트가 필요하다.
//
// ARTEX_PG_DSN 이 없으면 건너뛴다(저장소의 다른 DB 통합 테스트와 같은 규칙).
func TestLocalizeSeedAgentNames(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	t.Cleanup(func() { m.Close() })
	s := &Server{m: m}

	// 시드가 만드는 것과 같은 모양의 agent 를 직접 넣는다(전체 시드를 돌리지 않아도
	// 이 함수의 계약만 검증하면 충분하다).
	seed := func(key, name, desc string) {
		t.Helper()
		if _, err := m.pg.Exec(`DELETE FROM agents WHERE key=$1`, key); err != nil {
			t.Fatalf("정리 실패: %v", err)
		}
		if _, err := m.pg.Exec(
			`INSERT INTO agents(key,name,description,role,builtin,enabled) VALUES ($1,$2,$3,'assistant',false,true)`,
			key, name, desc); err != nil {
			t.Fatalf("시드 실패: %v", err)
		}
	}
	// locale 플래그를 지워 매번 "방금 언어가 바뀐" 상태로 만든다(전이를 관찰하기 위해).
	clearFlag := func() { _, _ = m.pg.Exec(`DELETE FROM settings WHERE key='seed_agent_names_locale'`) }
	nameOf := func(key string) string {
		t.Helper()
		var n string
		if err := m.pg.QueryRow(`SELECT name FROM agents WHERE key=$1`, key).Scan(&n); err != nil {
			t.Fatalf("%s 조회 실패: %v", key, err)
		}
		return n
	}
	t.Cleanup(func() {
		_, _ = m.pg.Exec(`DELETE FROM agents WHERE key IN ('reporter','retester')`)
		clearFlag()
	})

	t.Run("legacy 중국어 라벨을 ko 로 바꾼다", func(t *testing.T) {
		// 초기 구현이 실패했던 경로: legacy 중국어 → 목표 ko.
		seed("reporter", reporterAgentNameZh, reporterAgentDescriptionZhLegacy)
		clearFlag()
		t.Setenv("ARTEX_LOCALE", "ko")
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != reporterAgentName {
			t.Fatalf("reporter = %q, 기대 %q", got, reporterAgentName)
		}
	})

	t.Run("현재 중국어 라벨을 ko 로 바꾼다", func(t *testing.T) {
		seed("reporter", reporterAgentNameZh, reporterAgentDescriptionZh)
		clearFlag()
		t.Setenv("ARTEX_LOCALE", "ko")
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != reporterAgentName {
			t.Fatalf("reporter = %q, 기대 %q", got, reporterAgentName)
		}
	})

	t.Run("ko 라벨을 zh 로 바꾼다", func(t *testing.T) {
		seed("reporter", reporterAgentName, reporterAgentDescription)
		seed("retester", retesterAgentName, retesterAgentDescription)
		clearFlag()
		t.Setenv("ARTEX_LOCALE", "zh")
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != reporterAgentNameZh {
			t.Fatalf("reporter = %q, 기대 %q", got, reporterAgentNameZh)
		}
		if got := nameOf("retester"); got != retesterAgentNameZh {
			t.Fatalf("retester = %q, 기대 %q", got, retesterAgentNameZh)
		}
	})

	t.Run("사용자가 고친 이름은 보존한다", func(t *testing.T) {
		const custom = "내가 고친 보고서 에이전트"
		seed("reporter", custom, "사용자 설명")
		clearFlag()
		t.Setenv("ARTEX_LOCALE", "zh")
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != custom {
			t.Fatalf("사용자 지정 이름이 덮어써졌습니다: %q", got)
		}
	})

	t.Run("플래그가 현재 locale 이어도 상태가 어긋나면 다시 맞춘다", func(t *testing.T) {
		// 예전 구현이 이름을 못 바꾸고도 플래그를 남긴 적이 있다. 그때 만들어진 DB 는
		// "플래그는 ko 인데 이름은 중국어" 상태이므로, 플래그만 보고 건너뛰면 영영 안 고쳐진다.
		seed("reporter", reporterAgentNameZh, reporterAgentDescriptionZh)
		if _, err := m.pg.Exec(
			`INSERT INTO settings(key,value) VALUES ('seed_agent_names_locale','ko')
			 ON CONFLICT(key) DO UPDATE SET value='ko'`); err != nil {
			t.Fatal(err)
		}
		t.Setenv("ARTEX_LOCALE", "ko")
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != reporterAgentName {
			t.Fatalf("reporter = %q, 기대 %q", got, reporterAgentName)
		}
	})

	t.Run("같은 locale 로 다시 돌면 아무것도 바꾸지 않는다", func(t *testing.T) {
		seed("reporter", reporterAgentNameZh, reporterAgentDescriptionZh)
		clearFlag()
		t.Setenv("ARTEX_LOCALE", "zh")
		s.localizeSeedAgentNames()
		if got, _, _ := m.pg.GetSetting("seed_agent_names_locale"); got != "zh" {
			t.Fatalf("플래그 = %q, 기대 zh", got)
		}
		// locale 이 그대로면, 사용자가 이름을 고쳐도 건드리지 않는다.
		if _, err := m.pg.Exec(`UPDATE agents SET name='고친 이름' WHERE key='reporter'`); err != nil {
			t.Fatal(err)
		}
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != "고친 이름" {
			t.Fatalf("locale 이 그대로인데 이름이 바뀌었습니다: %q", got)
		}
	})
}

// TestConfigLocale 은 ARTEX_LOCALE 해석 규칙(미설정·미지원 값은 ko)을 지킨다.
func TestConfigLocale(t *testing.T) {
	cases := []struct{ env, want string }{
		{"", "ko"},
		{"ko", "ko"},
		{"zh", "zh"},
		{"ZH", "zh"},   // 대소문자 무시
		{" zh ", "zh"}, // 공백 무시
		{"en", "ko"},   // 미지원 → 기본값
		{"fr", "ko"},
	}
	for _, c := range cases {
		t.Setenv("ARTEX_LOCALE", c.env)
		if got := config.Locale(); got != c.want {
			t.Fatalf("ARTEX_LOCALE=%q → %q, 기대 %q", c.env, got, c.want)
		}
	}
}
