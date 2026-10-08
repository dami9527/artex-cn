package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// 本文件守住 F4 ④⑤（telegram·webhook·公共 HTTP 传输层）的简体中文不回归：
// telegram.go 的消息标签与汇总标题、webhook.go 的配置校验错误、http.go 的传输错误，
// 以及五个渠道共用的 validateHTTPURL，一旦回退就会被抓出来。辅助函数
// hasHan·hasHangul·assertChinese 位于 notify_localized_test.go（同包）。
//
// 数据（标题、类型、资产、摘要）全部保持 ASCII。这样「整份输出不含谚文」这条断言
// 才只盯骨架标签的回归，而不是被内容干扰。严重级别与状态标签在 F4 ① 已中文化，
// 输出里呈现为汉字，能通过断言。

// TestTelegramItemLabelsLocalized 打开单条 Telegram 消息的全部标签分支，
// 确认骨架是简体中文且不含谚文。
func TestTelegramItemLabelsLocalized(t *testing.T) {
	m := Message{
		HomeURL: "https://example.com/panel",
		Items: []Item{{
			Name:       "sqli-login", // Title() = Name
			VulnClass:  "injection",  // 必须与 Title() 不同才会渲染类型行
			Severity:   "high",       // SeverityLabel → 高危(汉字)
			Summary:    "login form is injectable",
			Assets:     []string{"host-a.example.com"},
			DetailURL:  "https://example.com/f/1",
			FromStatus: "pending", // IsStatusChange()=true → 渲染状态变更行
			ToStatus:   "fixed",
		}},
	}
	out, kept := telegramHTML(m)
	if kept != 1 {
		t.Fatalf("单条消息必须报告 1，实际得到 %d", kept)
	}
	assertChinese(t, "telegramHTML(single)", out)
	for _, want := range []string{"状态变更", "类型", "资产", "摘要", "查看详情"} {
		if !strings.Contains(out, want) {
			t.Errorf("Telegram 正文缺少 %q 标签:\n%s", want, out)
		}
	}
}

// TestTelegramBatchTitleLocalized 确认汇总(digest)标题的超量与时间窗分支渲染成
// 简体中文。必须与 markdownTitle 逐字一致，否则渠道之间会混杂。
func TestTelegramBatchTitleLocalized(t *testing.T) {
	items := []Item{{Name: "a", Severity: "high"}, {Name: "b", Severity: "low"}}

	// 有超量 + 有时间窗：汇总、总条数、前 N 条、其余、下一条、近 N 分钟。
	over := telegramBatchTitle(Message{WindowMinutes: 30}, items, 5)
	assertChinese(t, "telegramBatchTitle(over)", over)
	for _, want := range []string{"漏洞汇总 · 共 5 条", "显示前 2 条", "其余 3 条", "下一条继续", "近 30 分钟"} {
		if !strings.Contains(over, want) {
			t.Errorf("超量标题缺少 %q: %q", want, over)
		}
	}

	// 无超量 + 无时间窗：不应出现下一条提示。
	full := telegramBatchTitle(Message{}, items, 2)
	assertChinese(t, "telegramBatchTitle(full)", full)
	if !strings.Contains(full, "漏洞汇总 · 共 2 条") {
		t.Errorf("全量标题必须包含 '漏洞汇总 · 共 2 条': %q", full)
	}
	if strings.Contains(full, "下一条继续") {
		t.Errorf("未超量却写入了 '下一条继续' 提示: %q", full)
	}
}

// TestTelegramValidateLocalized 确认必填字段缺失的错误是简体中文，并说明缺的是哪个
// 字段。技术术语 Bot Token·Chat ID 原样保留。
func TestTelegramValidateLocalized(t *testing.T) {
	if err := (telegramChannel{}).Validate(map[string]any{}); err == nil {
		t.Fatal("缺少 bot_token 必须校验失败")
	} else {
		assertChinese(t, "telegram Validate(bot_token)", err.Error())
		if !strings.Contains(err.Error(), "Bot Token") {
			t.Errorf("必须提示缺少 Bot Token: %q", err.Error())
		}
	}
	if err := (telegramChannel{}).Validate(map[string]any{"bot_token": "t"}); err == nil {
		t.Fatal("缺少 chat_id 必须校验失败")
	} else {
		assertChinese(t, "telegram Validate(chat_id)", err.Error())
		if !strings.Contains(err.Error(), "Chat ID") {
			t.Errorf("必须提示缺少 Chat ID: %q", err.Error())
		}
	}
}

// TestWebhookValidateLocalized 确认通用 Webhook 配置校验的三类错误都是简体中文，
// 且每条错误都点明原因。
func TestWebhookValidateLocalized(t *testing.T) {
	if err := (webhookChannel{}).Validate(map[string]any{}); err == nil {
		t.Fatal("缺少 url 必须校验失败")
	} else {
		assertChinese(t, "webhook Validate(url)", err.Error())
		if !strings.Contains(err.Error(), "目标 URL") {
			t.Errorf("必须提示缺少目标 URL: %q", err.Error())
		}
	}
	// 不支持的方法：简体中文 + 暴露允许列表。
	if err := (webhookChannel{}).Validate(map[string]any{"url": "https://example.com/hook", "method": "DELETE"}); err == nil {
		t.Fatal("DELETE 必须校验失败")
	} else {
		assertChinese(t, "webhook Validate(method)", err.Error())
		if !strings.Contains(err.Error(), "GET/POST/PUT/PATCH") {
			t.Errorf("必须列出允许的方法: %q", err.Error())
		}
	}
	// 模板语法错误：简体中文（被包装的原因是 Go 模板错误，本身不含谚文）。
	if err := (webhookChannel{}).Validate(map[string]any{"url": "https://example.com/hook", "body_template": "{{"}); err == nil {
		t.Fatal("损坏的模板必须校验失败")
	} else {
		assertChinese(t, "webhook Validate(template)", err.Error())
		if !strings.Contains(err.Error(), "模板") {
			t.Errorf("必须说明是模板错误: %q", err.Error())
		}
	}
}

// TestValidateHTTPURLLocalized 确认五个渠道（telegram·webhook·dingtalk·feishu·wecom）
// 共用的 URL 校验辅助函数输出简体中文。这个辅助函数若留着其它语言，
// 就会与该函数自己的中文前缀（"API 地址无效: ..."）混在一起。
func TestValidateHTTPURLLocalized(t *testing.T) {
	// 不支持的协议。
	if err := validateHTTPURL("ftp://example.com/x"); err == nil {
		t.Fatal("ftp 协议必须被拒绝")
	} else {
		assertChinese(t, "validateHTTPURL(scheme)", err.Error())
		if !strings.Contains(err.Error(), "http") {
			t.Errorf("必须说明支持的协议(http/https): %q", err.Error())
		}
	}
	// 缺少主机名。
	if err := validateHTTPURL("http://"); err == nil {
		t.Fatal("没有主机的地址必须被拒绝")
	} else {
		assertChinese(t, "validateHTTPURL(host)", err.Error())
	}
}

// TestHTTPLocalTargetBlockedLocalized 确认本机/链路本地地址的拦截错误是简体中文，
// 并给出绕过方式（环境变量）。
func TestHTTPLocalTargetBlockedLocalized(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "") // 显式关闭，走拦截路径
	err := blockInternalDial("tcp", "127.0.0.1:25", nil)
	if err == nil {
		t.Fatal("本机地址默认必须被拦截")
	}
	assertChinese(t, "blockInternalDial", err.Error())
	if !strings.Contains(err.Error(), AllowLocalTargetsEnv) {
		t.Errorf("拦截错误必须说明绕过方式(%s): %q", AllowLocalTargetsEnv, err.Error())
	}
}

// TestHTTPRedactHelpersLocalized 确认地址脱敏(redact)占位符与传输错误的
// "未知错误" 分支是简体中文。
func TestHTTPRedactHelpersLocalized(t *testing.T) {
	if got := redactRequestTarget(""); !strings.Contains(got, "不可解析") {
		t.Errorf("无法解析的地址必须用中文占位符，实际得到 %q", got)
	}
	// *url.Error 的 Err 为 nil 的分支。
	got := redactTransportError(&url.Error{Op: "Get", URL: "http://api.example", Err: nil})
	if hasHangul(got) {
		t.Errorf("传输错误脱敏结果仍残留谚文: %q", got)
	}
	if !strings.Contains(got, "未知错误") {
		t.Errorf("Err 为 nil 时必须输出 '未知错误'，实际得到 %q", got)
	}
}

// TestHTTPStatusErrorsLocalized 通过真实的 doJSON 往返确认状态码分类（拒绝/服务端
// 异常/限流）暴露给用户的错误是简体中文。向 127.0.0.1 httptest 投递默认被拦截，
// 因此仅在本测试里显式放行。
func TestHTTPStatusErrorsLocalized(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "1")
	cases := []struct {
		code int
		want string
	}{
		{http.StatusForbidden, "对方拒绝请求"},
		{http.StatusInternalServerError, "对方服务异常"},
		{http.StatusTooManyRequests, "对方限流或超时"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte("body"))
		}))
		_, err := doJSON(context.Background(), http.MethodPost, srv.URL, nil, map[string]any{"a": 1})
		srv.Close()
		if err == nil {
			t.Fatalf("HTTP %d 必须报错", tc.code)
		}
		assertChinese(t, "doJSON(HTTP status)", err.Error())
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("HTTP %d 错误必须包含 %q，实际得到 %q", tc.code, tc.want, err.Error())
		}
	}
}
