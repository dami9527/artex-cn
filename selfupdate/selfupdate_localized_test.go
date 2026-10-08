package selfupdate

// 固定自更新路径上面向用户的进度与错误文案为简体中文的回归测试(F20)。
// server/update.go 的 updateHub 会把 Stage 的进度回调消息作为 SSE
// progress.message，把 Stage/FetchLatest/Rollback 返回的错误原样作为 progress.error、
// writeErr 正文、boot_notice 暴露给前端(update-card)，所以这些文案一旦回退，
// 用户界面就会中韩混杂。日志(log.Printf)与注释属于 Z2，不在范围内；
// 这里只断言「有汉字，且没有谚文」。

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// assertChineseError 断言错误存在、含汉字且不含谚文。
// 用 %w 包裹的底层 stdlib 错误里的英文是允许的（只禁谚文）。
func assertChineseError(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望有错误，实际是 nil", label)
	}
	msg := err.Error()
	if !hasHan(msg) {
		t.Errorf("%s: 没有汉字: %q", label, msg)
	}
	if hasHangul(msg) {
		t.Errorf("%s: 仍残留谚文: %q", label, msg)
	}
}

// rtFunc 用假响应注入 http.Client。不会真正访问 GitHub。
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubClient(status int, body string) *http.Client {
	return &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
}

// TestCheckURLErrorsLocalized: 原有的 TestCheckURLRejectsNonGitHub 只看是否拒绝，
// 不看文案语言。这里确认两条拒绝路径的错误都是简体中文。
func TestCheckURLErrorsLocalized(t *testing.T) {
	nonHTTPS := checkURL(mustParse(t, "http://api.github.com/x"))
	assertChineseError(t, "non-https", nonHTTPS)
	if !strings.Contains(nonHTTPS.Error(), "HTTPS") {
		t.Errorf("non-https 错误里没有 HTTPS 字样: %q", nonHTTPS.Error())
	}

	nonGitHub := checkURL(mustParse(t, "https://evil.example.com/x"))
	assertChineseError(t, "non-github-host", nonGitHub)
	if !strings.Contains(nonGitHub.Error(), "GitHub") {
		t.Errorf("non-github 错误里没有 GitHub 字样: %q", nonGitHub.Error())
	}
}

// TestFetchLatestErrorsLocalized: 确认版本检查路径的四类错误（限流、未发布、其它
// 状态码、解析失败、缺 tag）都是简体中文。用假 transport，不访问网络。
func TestFetchLatestErrorsLocalized(t *testing.T) {
	ctx := context.Background()

	_, errRate := FetchLatest(ctx, stubClient(http.StatusForbidden, ""))
	assertChineseError(t, "rate-limited", errRate)
	if !strings.Contains(errRate.Error(), "60") {
		t.Errorf("限流错误里没有 60 字样: %q", errRate.Error())
	}

	_, errNF := FetchLatest(ctx, stubClient(http.StatusNotFound, ""))
	assertChineseError(t, "not-found", errNF)

	_, errStatus := FetchLatest(ctx, stubClient(http.StatusInternalServerError, ""))
	assertChineseError(t, "bad-status", errStatus)
	if !strings.Contains(errStatus.Error(), "GitHub") {
		t.Errorf("状态错误里没有 GitHub 字样: %q", errStatus.Error())
	}

	_, errJSON := FetchLatest(ctx, stubClient(http.StatusOK, "{not json"))
	assertChineseError(t, "bad-json", errJSON)

	_, errTag := FetchLatest(ctx, stubClient(http.StatusOK, "{}"))
	assertChineseError(t, "missing-tag", errTag)
	if !strings.Contains(errTag.Error(), "tag") {
		t.Errorf("缺 tag 错误里没有 tag 字样: %q", errTag.Error())
	}
}

// TestStageMissingAssetLocalized: 发布里没有当前平台包时，Stage 会在下载之前
// 以简体中文错误停下（FindAsset 失败 → 不访问网络）。视运行环境，checkWritable
// 也可能先拦下，不过那个错误同样是中文，所以这里只断言语言。
func TestStageMissingAssetLocalized(t *testing.T) {
	rel := &Release{TagName: "v9.9.9"} // Assets 为空
	err := Stage(context.Background(), &http.Client{}, rel, "0.0.1", nil)
	assertChineseError(t, "stage-missing-asset", err)
}
