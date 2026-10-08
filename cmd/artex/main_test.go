package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/agent"
)

func TestShutdownContextPreservesNamedCause(t *testing.T) {
	signalCtx, signalCancel := context.WithCancel(context.Background())
	ctx, shutdown := shutdownContext(signalCtx)
	defer shutdown(nil)
	signalCancel()
	<-ctx.Done()
	if code, _, _, ok := agent.AbortReason(ctx); !ok || code != "shutdown" {
		t.Fatalf("code=%q ok=%v, want shutdown", code, ok)
	}
}

// TestPrintBannerLocalized 验证打印到标准输出的启动横幅是简体中文，且不残留谚文。
// 这是用户运行 ARTEX 看到的第一屏内容。run() 里的 "[config] ..." 日志行有意不在
// 本用例范围内——它们属于日志，在本地化优先级里排最后（backlog Z2）。
func TestPrintBannerLocalized(t *testing.T) {
	out := captureStdout(t, func() { printBanner(":8787") })
	t.Logf("横幅样例:\n%s", out)

	hasHan := strings.ContainsFunc(out, func(r rune) bool {
		return unicode.Is(unicode.Han, r)
	})
	if !hasHan {
		t.Fatalf("横幅里没有汉字: %q", out)
	}

	for _, want := range []string{"AI 自主渗透测试系统", "版本", "监听", ":8787"} {
		if !strings.Contains(out, want) {
			t.Errorf("横幅缺少 %q: %q", want, out)
		}
	}

	for _, r := range out {
		if unicode.Is(unicode.Hangul, r) {
			t.Errorf("横幅残留谚文(%q): %q", r, out)
		}
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns whatever
// fn wrote. printBanner uses fmt.Print*, which resolves os.Stdout at call time,
// so swapping it here captures the banner.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}
