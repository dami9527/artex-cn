package db

import (
	"testing"
	"unicode"
)

// assertRetestReasonChinese 断言复测原因常量含汉字且不含谚文（F9）。
// 这些常量会拼进 FinishFindingRetest·RecoverFindingRetests 的 SQL 字面量位置，
// 钉住它们也就同时保护了面板上暴露给用户的文案。
func assertRetestReasonChinese(t *testing.T, name, s string) {
	t.Helper()
	if s == "" {
		t.Fatalf("%s: 空字符串", name)
	}
	hasHan := false
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s: 仍残留谚文: %q", name, s)
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	if !hasHan {
		t.Fatalf("%s: 没有汉字: %q", name, s)
	}
}

// TestFindingRetestReasonsLocalized 断言存入 finding_retests.error 列、并在复测面板
// (finding-retest-panel) 以 item.error 暴露的两个终结原因常量是简体中文。
// 它们与 server/conversations.go 的同类原因(convRetest*)共用同一列与同一面板，
// 只要有一个不是中文，同一个面板里就会混语言。
func TestFindingRetestReasonsLocalized(t *testing.T) {
	assertRetestReasonChinese(t, "retestNoConclusionReason", retestNoConclusionReason)
	assertRetestReasonChinese(t, "retestServiceRestartReason", retestServiceRestartReason)
}
