package server

import (
	"testing"

	"github.com/Autumn-27/artex/config"
)

// TestLocalizeSeedAgentNames 用真实 DB 验证 localizeSeedAgentNames 的契约。
//
// 需要这个测试的原因: 种子标签以字符串写进 DB，而种子只跑一次，因此
// ① 文案改动后已安装实例的名称·描述仍是旧值，② 每次启动都覆盖则会丢失
// 用户在 UI 上改过的名字。localizeSeedAgentNames 就是来兜住这两者的 ——
// 仅当当前值是该智能体历代默认值之一(= 用户未改动过)时才对齐到当前文案。
// 静态测试(reporter_seed_localized_test.go)只看常量，抓不到 DB 状态迁移 ——
// 那种「标志已是当前 locale、实际值却还是旧文案」的错位只能由这个测试发现。
//
// 本仓库固定为中文(config.Locale() 一律返回 "zh")，因此这里不再切换语言，
// 只断言最终名称·描述是中文且不含谚文。
//
// 没有 ARTEX_PG_DSN 时跳过(与仓库里其他 DB 集成测试同一规则)。
func TestLocalizeSeedAgentNames(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	t.Cleanup(func() { m.Close() })
	s := &Server{m: m}

	// 直接插入与种子形状相同的 agent(不必跑完整种子，
	// 只验证这个函数的契约就够了)。
	seed := func(key, name, desc string) {
		t.Helper()
		if _, err := m.pg.Exec(`DELETE FROM agents WHERE key=$1`, key); err != nil {
			t.Fatalf("清理失败: %v", err)
		}
		if _, err := m.pg.Exec(
			`INSERT INTO agents(key,name,description,role,builtin,enabled) VALUES ($1,$2,$3,'assistant',false,true)`,
			key, name, desc); err != nil {
			t.Fatalf("种子写入失败: %v", err)
		}
	}
	// 清掉 locale 标志，每次都造成「刚刚对齐过」的状态(以便观察迁移)。
	clearFlag := func() { _, _ = m.pg.Exec(`DELETE FROM settings WHERE key='seed_agent_names_locale'`) }
	nameOf := func(key string) string {
		t.Helper()
		var n string
		if err := m.pg.QueryRow(`SELECT name FROM agents WHERE key=$1`, key).Scan(&n); err != nil {
			t.Fatalf("%s 查询失败: %v", key, err)
		}
		return n
	}
	descOf := func(key string) string {
		t.Helper()
		var d string
		if err := m.pg.QueryRow(`SELECT description FROM agents WHERE key=$1`, key).Scan(&d); err != nil {
			t.Fatalf("%s 查询失败: %v", key, err)
		}
		return d
	}
	t.Cleanup(func() {
		_, _ = m.pg.Exec(`DELETE FROM agents WHERE key IN ('reporter','retester')`)
		clearFlag()
	})

	t.Run("把历史描述对齐到当前中文标签", func(t *testing.T) {
		// 早期实现失败过的路径: 历史文案 → 当前文案。
		seed("reporter", reporterAgentName, reporterAgentDescriptionLegacy)
		clearFlag()
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != reporterAgentName {
			t.Fatalf("reporter = %q, 期望 %q", got, reporterAgentName)
		}
		if got := descOf("reporter"); got != reporterAgentDescription {
			t.Fatalf("reporter description = %q, 期望 %q", got, reporterAgentDescription)
		}
		assertChineseError(t, "reporter_name", nameOf("reporter"))
		assertChineseError(t, "reporter_description", descOf("reporter"))
	})

	t.Run("保留用户改过的名字与描述", func(t *testing.T) {
		const custom = "我改过的报告智能体"
		const customDesc = "用户说明"
		seed("reporter", custom, customDesc)
		clearFlag()
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != custom {
			t.Fatalf("用户自定义名称被覆盖了: %q", got)
		}
		if got := descOf("reporter"); got != customDesc {
			t.Fatalf("用户自定义描述被覆盖了: %q", got)
		}
	})

	t.Run("标志已是当前 locale 但状态不一致时重新对齐", func(t *testing.T) {
		// 旧实现有过改不动名字却留下标志的情况。那时生成的 DB 处于
		// 「标志已是当前 locale、实际值还是旧文案」的状态，只看标志跳过就永远修不好。
		seed("reporter", reporterAgentName, reporterAgentDescriptionLegacy)
		if _, err := m.pg.Exec(
			`INSERT INTO settings(key,value) VALUES ('seed_agent_names_locale','zh')
			 ON CONFLICT(key) DO UPDATE SET value='zh'`); err != nil {
			t.Fatal(err)
		}
		s.localizeSeedAgentNames()
		if got := descOf("reporter"); got != reporterAgentDescription {
			t.Fatalf("reporter description = %q, 期望 %q", got, reporterAgentDescription)
		}
	})

	t.Run("标志与状态都一致时不做任何改动", func(t *testing.T) {
		seed("reporter", reporterAgentName, reporterAgentDescription)
		clearFlag()
		s.localizeSeedAgentNames()
		if got, _, _ := m.pg.GetSetting("seed_agent_names_locale"); got != "zh" {
			t.Fatalf("标志 = %q, 期望 zh", got)
		}
		// 标志一致时，用户改过的名字也不会被碰。
		if _, err := m.pg.Exec(`UPDATE agents SET name='改过的名字' WHERE key='reporter'`); err != nil {
			t.Fatal(err)
		}
		s.localizeSeedAgentNames()
		if got := nameOf("reporter"); got != "改过的名字" {
			t.Fatalf("标志一致时名字却被改了: %q", got)
		}
	})
}

// TestConfigLocale 断言 ARTEX_LOCALE 无论取何值(含未设置)都回落到 "zh"：
// 本仓库界面固定为简体中文，不存在其它语言的 locale 分支。
func TestConfigLocale(t *testing.T) {
	cases := []struct{ env, want string }{
		{"", "zh"},
		{"zh", "zh"},
		{"ZH", "zh"},   // 忽略大小写
		{" zh ", "zh"}, // 忽略空白
		{"en", "zh"},   // 未支持 → 回落
		{"ko", "zh"},   // 未支持 → 回落
		{"fr", "zh"},   // 未支持 → 回落
	}
	for _, c := range cases {
		t.Setenv("ARTEX_LOCALE", c.env)
		if got := config.Locale(); got != c.want {
			t.Fatalf("ARTEX_LOCALE=%q → %q, 期望 %q", c.env, got, c.want)
		}
	}
}
