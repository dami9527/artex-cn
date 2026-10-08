package llmpool

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode"

	"github.com/Autumn-27/norma/llm"
)

// hasHangul 判断 s 是否含有谚文字符。
func hasHangul(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

// hasHan 判断 s 是否含有 CJK（汉字）字符。
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// assertSurfacedExhaustion 检查整条链耗尽后交回的错误：它必须仍然包裹 ErrExhausted
// 哨兵（调用方依赖 errors.Is）、是可读的简体中文（含汉字、用全角冒号），并以本地化后的
// 哨兵文案开头。failProv 的内容是 ASCII（"anthropic: status 402: nope"），
// 所以最终字符串里的汉字只能来自本地化哨兵。
func assertSurfacedExhaustion(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 耗尽链返回了 nil 错误", label)
	}
	if !errors.Is(err, ErrExhausted) {
		t.Fatalf("%s: 交回的错误不再包裹 ErrExhausted: %v", label, err)
	}
	msg := err.Error()
	if !hasHan(msg) {
		t.Errorf("%s: 交回的错误不含汉字: %q", label, msg)
	}
	if hasHangul(msg) {
		t.Errorf("%s: 交回的错误仍残留谚文: %q", label, msg)
	}
	if !strings.ContainsRune(msg, '：') {
		t.Errorf("%s: 交回的错误没有使用全角冒号: %q", label, msg)
	}
	if !strings.HasPrefix(msg, ErrExhausted.Error()) {
		t.Errorf("%s: 交回的错误没有以本地化哨兵文案开头: %q", label, msg)
	}
}

// 链耗尽错误是面向用户的：整条 LLM 链都失败的 worker 会经 agent/capture.go 把它
// 记为 "result" 活动，展示在运行记录里。所以它的文案必须是简体中文，
// 同时其身份（errors.Is）必须保留，供按哨兵分支的调用方使用。
func TestExhaustedErrorLocalized(t *testing.T) {
	assertChineseErrText(t, "ErrExhausted", ErrExhausted.Error())

	// 两条上报路径都走一遍，未来改动任一 wrap 位置都会被抓到。
	streamChain := New([]*Member{
		member(1, "a", 10, failProv("a", 402)),
		member(2, "b", 5, failProv("b", 402)),
	}, NewRegistry(nil, nil))
	_, serr := drain(streamChain.Stream(context.Background(), llm.CompletionRequest{}))
	assertSurfacedExhaustion(t, "Stream", serr)

	completeChain := New([]*Member{
		member(1, "a", 10, failProv("a", 402)),
		member(2, "b", 5, failProv("b", 402)),
	}, NewRegistry(nil, nil))
	_, _, _, cerr := completeChain.Complete(context.Background(), llm.CompletionRequest{})
	assertSurfacedExhaustion(t, "Complete", cerr)
}

// assertChineseErrText 在 s 不含汉字或残留谚文时失败。
func assertChineseErrText(t *testing.T, label, s string) {
	t.Helper()
	if !hasHan(s) {
		t.Errorf("%s: 没有汉字: %q", label, s)
	}
	if hasHangul(s) {
		t.Errorf("%s: 仍残留谚文: %q", label, s)
	}
}
