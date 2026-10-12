package server

import (
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/config"
)

// TestTrMsgSwitchesLanguage 钉住后端输出语言切换：用户可见的中文原文在中文（以及钩子
// 未注入）时原样返回，在 en/es 下翻译成不含汉字的外文串。这是 web 端 i18n 目录在后端的
// 对应物——它证明 writeErr 的响应遵循配置的语言，而不是永远只出中文。
func TestTrMsgSwitchesLanguage(t *testing.T) {
	t.Cleanup(func() { config.CurrentLanguage = nil })

	hasHangul := func(s string) bool {
		for _, r := range s {
			if unicode.Is(unicode.Hangul, r) {
				return true
			}
		}
		return false
	}
	hasHan := func(s string) bool {
		for _, r := range s {
			if unicode.Is(unicode.Han, r) {
				return true
			}
		}
		return false
	}

	// 一个有代表性的目录键（服务器错误常量的真实取值）。
	const zhMsg = errAssetIDRequired
	if !hasHan(zhMsg) {
		t.Fatalf("前置条件：源消息应当是中文，实际得到 %q", zhMsg)
	}
	if hasHangul(zhMsg) {
		t.Fatalf("前置条件：源消息不得含谚文，实际得到 %q", zhMsg)
	}

	// 未注入钩子 → ActiveLanguage 回落 "zh" → 原样返回。
	config.CurrentLanguage = nil
	if got := trMsg(zhMsg); got != zhMsg {
		t.Fatalf("未注入钩子：trMsg 必须原样返回中文原文，实际得到 %q", got)
	}

	// zh → 原样返回（键本身就是中文原文，目录里不重复存 zh）。
	config.CurrentLanguage = func() string { return "zh" }
	if got := trMsg(zhMsg); got != zhMsg {
		t.Fatalf("zh：trMsg 必须返回中文原文，实际得到 %q", got)
	}

	// en/es → 与中文原文不同的非空译文，且不得残留汉字或谚文。
	for _, lang := range []string{"en", "es"} {
		config.CurrentLanguage = func() string { return lang }
		got := trMsg(zhMsg)
		if got == "" || got == zhMsg {
			t.Fatalf("%s：trMsg 必须翻译成与中文原文不同的内容，实际得到 %q", lang, got)
		}
		if hasHangul(got) || hasHan(got) {
			t.Fatalf("%s：译文不得含汉字或谚文，实际得到 %q", lang, got)
		}
	}

	// 未知语言代码 → 回落到中文原文（config.NormalizeLanguage 会拒绝它）。
	config.CurrentLanguage = func() string { return "fr" }
	if got := trMsg(zhMsg); got != zhMsg {
		t.Fatalf("未知语言：trMsg 必须回落到中文原文，实际得到 %q", got)
	}

	// 目录里没有的字符串（下层传播上来的错误）原样透传。
	config.CurrentLanguage = func() string { return "en" }
	const notInCatalog = "some-lower-layer-error: connection refused"
	if got := trMsg(notInCatalog); got != notInCatalog {
		t.Fatalf("不在目录里的字符串必须原样透传，实际得到 %q", got)
	}
}
