package notify

import (
	"strings"
	"testing"
)

// 本文件守住 F4 ③（邮件正文、HTML 模板）的简体中文不回归：
// html.go 的标签与 email.go 的校验错误一旦回退就会被抓出来。
// 辅助函数 hasHan·hasHangul·assertChinese 位于 notify_localized_test.go（同包）。
//
// 数据（标题、类型、资产、摘要）全部保持 ASCII。这样「整份输出不含谚文」这条断言
// 才只盯骨架标签的回归，而不是被内容干扰。严重级别与状态标签在 F4 ① 已中文化，
// 输出里呈现为汉字，能通过断言。

// TestHTMLItemLabelsLocalized 打开单条邮件正文的全部标签分支，
// 确认骨架是简体中文且不含谚文。
func TestHTMLItemLabelsLocalized(t *testing.T) {
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
	out := htmlBody(m, 0)
	assertChinese(t, "htmlBody(single)", out)
	for _, want := range []string{
		"状态变更", "类型", "资产", "摘要", "查看详情", "在平台中查看全部",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("邮件正文缺少 %q 标签:\n%s", want, out)
		}
	}
}

// TestHTMLBatchIntroLocalized 确认汇总(digest)导读的两个分支（有无时间窗）
// 都渲染成简体中文。
func TestHTMLBatchIntroLocalized(t *testing.T) {
	items := []Item{{Name: "a", Severity: "high"}, {Name: "b", Severity: "low"}}

	withWindow := htmlBatchIntro(Message{Batch: true, WindowMinutes: 30, Items: items})
	assertChinese(t, "htmlBatchIntro(有时间窗)", withWindow)
	for _, want := range []string{"近 30 分钟", "新增", "2 个漏洞"} {
		if !strings.Contains(withWindow, want) {
			t.Errorf("时间窗导读缺少 %q: %q", want, withWindow)
		}
	}

	noWindow := htmlBatchIntro(Message{Batch: true, WindowMinutes: 0, Items: items})
	assertChinese(t, "htmlBatchIntro(无时间窗)", noWindow)
	if !strings.Contains(noWindow, "新增 2 个漏洞") {
		t.Errorf("无时间窗导读必须包含 %q: %q", "新增 2 个漏洞", noWindow)
	}
	if strings.Contains(noWindow, "分钟") {
		t.Errorf("没有时间窗却写入了 '分钟': %q", noWindow)
	}
}

// TestEmailValidateLocalized 确认 SMTP 配置校验的四类错误都是简体中文，
// 且每条错误都点明是哪个字段有问题。这些消息会直接展示给配置通知渠道的用户。
func TestEmailValidateLocalized(t *testing.T) {
	cases := []struct {
		name   string
		cfg    map[string]any
		substr string
	}{
		{"缺少服务器地址", map[string]any{"port": float64(25), "from": "a@b.c", "to": []any{"d@e.f"}}, "SMTP"},
		{"端口超出范围", map[string]any{"host": "h"}, "端口"},
		{"缺少发件人", map[string]any{"host": "h", "port": float64(25), "to": []any{"d@e.f"}}, "发件人"},
		{"缺少收件人", map[string]any{"host": "h", "port": float64(25), "from": "a@b.c"}, "收件人"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (emailChannel{}).Validate(tc.cfg)
			if err == nil {
				t.Fatalf("校验必须失败: %v", tc.cfg)
			}
			assertChinese(t, "Validate("+tc.name+")", err.Error())
			if !strings.Contains(err.Error(), tc.substr) {
				t.Errorf("错误消息必须包含 %q，实际得到 %q", tc.substr, err.Error())
			}
		})
	}
}
