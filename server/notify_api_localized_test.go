package server

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防御测试：保证 notify_api.go 的通知配置 API 响应文案与测试消息为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。

// TestNotifyErrorConstantsLocalized 断言 10 个响应常量与 3 个测试消息常量
// 全部为简体中文。任何一处改回旧文案，本测试即失败。
func TestNotifyErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"notifyErrBadJSON":         notifyErrBadJSON,
		"notifyErrKindInvalidFmt":  fmt.Sprintf(notifyErrKindInvalidFmt, "email / webhook"),
		"notifyErrNameMissing":     notifyErrNameMissing,
		"notifyErrNameEmpty":       notifyErrNameEmpty,
		"notifyErrModeInvalid":     notifyErrModeInvalid,
		"notifyErrRateNegative":    notifyErrRateNegative,
		"notifyErrChannelID":       notifyErrChannelID,
		"notifyErrKindUnregFmt":    fmt.Sprintf(notifyErrKindUnregFmt, "bogus"),
		"notifyErrDeliveryID":      notifyErrDeliveryID,
		"notifyErrChannelNotFound": notifyErrChannelNotFound,
		"notifyTestName":           notifyTestName,
		"notifyTestClass":          notifyTestClass,
		"notifyTestSummary":        notifyTestSummary,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestNotifyChannelLookupErrLocalized 用真实代码检查「渠道不存在 → 404」路径。
// notifyChannelLookupErr 是只靠 w 与 err 运行、不需要 Server/DB 的包级函数，因此把哨兵
// 错误原样传入时，无需 DB 即可确认中文常量确实进入 404 响应正文。
func TestNotifyChannelLookupErrLocalized(t *testing.T) {
	rec := httptest.NewRecorder()
	notifyChannelLookupErr(rec, db.ErrNotificationChannelNotFound)
	if rec.Code != 404 {
		t.Fatalf("状态码 = %d, 期望 = 404 (正文 %q)", rec.Code, rec.Body.String())
	}
	got := decodeErrorField(t, rec.Body.Bytes())
	if got != notifyErrChannelNotFound {
		t.Fatalf("响应文案 = %q, 期望 = %q", got, notifyErrChannelNotFound)
	}
	assertChineseError(t, "channel_not_found", got)
}

// TestNotifyTestMessageLocalized 用纯函数组装渠道连通性检查用的测试消息，
// 检查发往用户的正文(标题·分类·摘要)为简体中文。不需要 DB·网络。
func TestNotifyTestMessageLocalized(t *testing.T) {
	const base = "https://artex.example.test"
	msg := notifyTestMessage(base)
	if len(msg.Items) != 1 {
		t.Fatalf("测试消息条目数 = %d, 期望 = 1", len(msg.Items))
	}
	item := msg.Items[0]
	assertChineseError(t, "test.name", item.Name)
	assertChineseError(t, "test.class", item.VulnClass)
	assertChineseError(t, "test.summary", item.Summary)
	if msg.HomeURL != base || item.DetailURL != base {
		t.Fatalf("链接保留失败: HomeURL=%q DetailURL=%q (期望 %q)", msg.HomeURL, item.DetailURL, base)
	}
}
