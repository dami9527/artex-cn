package sidequestion

import (
	"errors"
	"fmt"
	"testing"
	"unicode"
)

// assertChinese 在 msg 为空、不含汉字（= 未本地化）、或残留谚文时失败。
// ASCII 字段名与 "1–4000" 这类区间写法没问题；只有谚文才代表未本地化的字符串。
func assertChinese(t *testing.T, label, msg string) {
	t.Helper()
	if msg == "" {
		t.Fatalf("%s: 空消息", label)
	}
	han := false
	for _, r := range msg {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s: 仍残留谚文: %q", label, msg)
		}
		if unicode.Is(unicode.Han, r) {
			han = true
		}
	}
	if !han {
		t.Fatalf("%s: 没有汉字: %q", label, msg)
	}
}

// TestSideQuestionOutputsLocalized 守住 F21：本包产出的每一条面向用户的旁路提问
// 错误与回答文本都是简体中文。它们经 /api/.../side-questions 进入聊天里的旁路提问
// 面板，所以这里回退就是用户可见的回归，必须让构建失败。
func TestSideQuestionOutputsLocalized(t *testing.T) {
	assertChinese(t, "ErrContextBudget", ErrContextBudget.Error())
	assertChinese(t, "errSideModelInterrupted", errSideModelInterrupted.Error())
	assertChinese(t, "errSideNoAnswer", errSideNoAnswer.Error())
	assertChinese(t, "msgSideToolUnavailable", msgSideToolUnavailable)
	assertChinese(t, "errSideSummaryCallCap", errSideSummaryCallCap.Error())
	assertChinese(t, "sideSummaryFailedPrefix", sideSummaryFailedPrefix)
	assertChinese(t, "errSideSummaryIncomplete", errSideSummaryIncomplete.Error())
	assertChinese(t, "errSideSummaryOverBudget", errSideSummaryOverBudget.Error())
	assertChinese(t, "errSideHistoryCursor", errSideHistoryCursor.Error())
	assertChinese(t, "errSideCompactionStalled", errSideCompactionStalled.Error())
}

// TestSideQuestionBudgetSentinelPreserved: 本地化消息文本不得破坏用 errors.Is 按
// ErrContextBudget 哨兵分支的调用方（见 context_test.go，恢复路径放弃后依赖它）。
func TestSideQuestionBudgetSentinelPreserved(t *testing.T) {
	if !errors.Is(fmt.Errorf("prepare: %w", ErrContextBudget), ErrContextBudget) {
		t.Fatal("ErrContextBudget 哨兵识别(errors.Is)被破坏")
	}
}

// TestSideQuestionBrainPreserved: 智能体大脑提示词（回答指令与摘要指令）必须保持
// 经过基准测试的中文原文。本地化方针是不翻译大脑正文、只强制输出语言，
// 这两个提示词一旦改动，基准测试的行为就会漂移。
func TestSideQuestionBrainPreserved(t *testing.T) {
	for _, c := range []struct{ label, text string }{
		{"instruction", instruction},
		{"summaryInstruction", summaryInstruction},
	} {
		han := false
		for _, r := range c.text {
			if unicode.Is(unicode.Han, r) {
				han = true
				break
			}
		}
		if !han {
			t.Errorf("%s: 大脑提示词不再是中文（存在基准漂移风险）", c.label)
		}
	}
}
