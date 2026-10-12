package config

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestStartupMessageSwitchesLanguage 钉住「数据库配置启动错误与来源标签」跟随
// ARTEX_LANG：默认中文，配置后可用英语／西班牙语。运维必须据以行动的标识符
// （ARTEX_PG_DSN、database、dsn、host/user/dbname）保持原样不译。
func TestStartupMessageSwitchesLanguage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARTEX_PG_DSN", "")
	t.Setenv("ARTEX_CONFIG", filepath.Join(dir, "nope.json"))

	// 默认（未设 ARTEX_LANG）→ 中文（artex-cn 的默认语言）。
	t.Setenv("ARTEX_LANG", "")
	assertChinese(t, "default startup error", mustDSNErr(t))

	// 显式中文 → 中文。
	t.Setenv("ARTEX_LANG", "zh")
	assertChinese(t, "zh startup error", mustDSNErr(t))

	// 英语 → 不含汉字，且保留标识符。
	t.Setenv("ARTEX_LANG", "en")
	en := mustDSNErr(t)
	if hasHan(en) {
		t.Errorf("en startup error should have no Han ideograph: %q", en)
	}
	for _, id := range []string{"ARTEX_PG_DSN", "database", "dsn", "host/user/dbname"} {
		if !containsStr(en, id) {
			t.Errorf("en startup error must keep identifier %q: %q", id, en)
		}
	}

	// 西班牙语 → 非空，且不得含汉字（说明确实走了 es 译文而不是回落中文）。
	t.Setenv("ARTEX_LANG", "es")
	if es := mustDSNErr(t); es == "" || hasHan(es) {
		t.Errorf("es startup error must be non-empty and non-Chinese: %q", es)
	}
}

func mustDSNErr(t *testing.T) string {
	t.Helper()
	_, _, err := PostgresDSN()
	if err == nil {
		t.Fatal("missing config should error")
	}
	return err.Error()
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestCatalogFormatVerbsMatch 钉住每一条译文都按同样顺序保留中文原文的格式化动词。
// 调用点只包裹模板本身（log.Printf(T(src), args...)），因此漏掉或调换动词都会在运维
// 控制台里打出 %!s(MISSING) 或者把参数对错位置。
func TestCatalogFormatVerbsMatch(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)
	for name, cat := range map[string]map[string]map[string]string{"cfgCatalog": cfgCatalog, "logCatalog": logCatalog} {
		for src, tr := range cat {
			want := strings.Join(verbs.FindAllString(src, -1), " ")
			for lang, v := range tr {
				if got := strings.Join(verbs.FindAllString(v, -1), " "); got != want {
					t.Errorf("%s[%q][%s] verbs = %q, want %q", name, src, lang, got, want)
				}
			}
		}
	}
}

// TestCatalogKeysAreChineseAndTranslated 钉住目录本身的两条不变量：
//  1. 键是中文原文（本仓库的源语言），不得出现谚文——上游以韩文原文为键的做法
//     在这里被改写成中文键，键里残留韩文就意味着调用点永远查不到译文。
//  2. 每条都有 en 与 es 译文，且译文里没有汉字（否则等于没翻译）。
func TestCatalogKeysAreChineseAndTranslated(t *testing.T) {
	for name, cat := range map[string]map[string]map[string]string{"cfgCatalog": cfgCatalog, "logCatalog": logCatalog} {
		for src, tr := range cat {
			if hasHangul(src) {
				t.Errorf("%s: key still contains Hangul: %q", name, src)
			}
			if !hasHan(src) {
				t.Errorf("%s: key should be Chinese source text: %q", name, src)
			}
			for _, lang := range []string{"en", "es"} {
				v, ok := tr[lang]
				if !ok || v == "" {
					t.Errorf("%s[%q]: missing %s translation", name, src, lang)
					continue
				}
				if lang == "en" && hasHan(v) {
					t.Errorf("%s[%q][en]: translation still contains Han ideographs: %q", name, src, v)
				}
			}
		}
	}
}
