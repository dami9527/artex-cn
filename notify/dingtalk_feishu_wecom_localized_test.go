package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// koFreeStatusChangeItem 构造不含谚文（ASCII）的状态变更条目。
// 标签若回退成谚文，渲染结果里就会出现谚文，因此只有数据本身谚文为 0 时，
// 「整份输出谚文为 0」这条断言才能精确抓到标签回退。
func koFreeStatusChangeItem() Item {
	return Item{
		FindingID:  7,
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/7",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("卡片序列化失败: %v", err)
	}
	return string(b)
}

// TestFeishuItemLinesLocalized 检查飞书卡片单条正文的字段标签（状态变更、类型、
// 资产、摘要）是简体中文，且与 markdown.go(F4②) 逐字一致。feishu.go 用的是这套
// 标签的自有副本（不是 markdown 三渠道共享函数），所以除了共享函数的回归，
// 还需要这个钉子单独盯住飞书正文回退的情况。
func TestFeishuItemLinesLocalized(t *testing.T) {
	got := feishuItemLines(koFreeStatusChangeItem())
	assertChinese(t, "feishuItemLines", got)
	for _, want := range []string{
		"**状态变更**：待处理 → 已修复",
		"**类型**：SQLi",
		"**资产**：a.example.com",
		"**摘要**：SQL injection via q param",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("飞书单条正文必须包含 %q:\n%s", want, got)
		}
	}
}

// TestFeishuCardButtonsLocalized 检查飞书卡片按钮标签（查看详情、在平台中查看全部）
// 是否为简体中文，并确认整张卡片没有谚文（数据本身为 ASCII）。
func TestFeishuCardButtonsLocalized(t *testing.T) {
	// 单条："查看详情" 按钮（DetailURL 非空时）
	single, _ := feishuCard(Message{Items: []Item{koFreeStatusChangeItem()}})
	singleJSON := mustJSON(t, single)
	if hasHangul(singleJSON) {
		t.Errorf("单条飞书卡片仍残留谚文:\n%s", singleJSON)
	}
	if !strings.Contains(singleJSON, "查看详情") {
		t.Errorf("单条飞书卡片必须有 '查看详情' 按钮:\n%s", singleJSON)
	}

	// 多条："在平台中查看全部" 按钮（HomeURL 非空时）
	batch, _ := feishuCard(Message{Batch: true, HomeURL: "https://platform.example", Items: koFreeItems(2)})
	batchJSON := mustJSON(t, batch)
	if hasHangul(batchJSON) {
		t.Errorf("多条飞书卡片仍残留谚文:\n%s", batchJSON)
	}
	if !strings.Contains(batchJSON, "在平台中查看全部") {
		t.Errorf("多条飞书卡片必须有 '在平台中查看全部' 按钮:\n%s", batchJSON)
	}
}

// TestDingTalkActionCardButtonLocalized 通过真实 Send 路径（假接收端）检查钉钉
// ActionCard 的 "查看详情" 按钮（singleTitle）是否为简体中文。
func TestDingTalkActionCardButtonLocalized(t *testing.T) {
	var singleTitle string
	srv := capturePost(t, `{"errcode":0,"errmsg":"ok"}`, func(_ *testing.T, body map[string]any, _ *http.Request) {
		card, _ := body["actionCard"].(map[string]any)
		singleTitle, _ = card["singleTitle"].(string)
	})
	m := Message{Items: []Item{koFreeStatusChangeItem()}}
	if _, err := (dingTalkChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, m); err != nil {
		t.Fatalf("投递失败: %v", err)
	}
	if singleTitle != "查看详情" {
		t.Errorf("钉钉 ActionCard singleTitle 必须是 '查看详情'，实际得到 %q", singleTitle)
	}
}

// TestChinaPlatformValidateLocalized 检查钉钉·飞书·企业微信的配置校验错误是简体中文，
// 且保留 "Webhook" 字样（通知配置保存与测试接口会把错误暴露给用户）。
func TestChinaPlatformValidateLocalized(t *testing.T) {
	for _, kind := range []string{KindDingTalk, KindFeishu, KindWeCom} {
		ch, ok := Get(kind)
		if !ok {
			t.Fatalf("%s 渠道未注册", kind)
		}

		// 空配置："缺少 Webhook 地址"
		missing := ch.Validate(map[string]any{})
		if missing == nil {
			t.Fatalf("%s: 空配置必须校验失败", kind)
		}
		assertChinese(t, kind+" missing", missing.Error())
		if !strings.Contains(missing.Error(), "Webhook") || !strings.Contains(missing.Error(), "缺少") {
			t.Errorf("%s: 缺失错误必须是 '缺少 Webhook 地址'，实际得到 %q", kind, missing)
		}

		// 非法地址(ftp)："Webhook 地址无效: …"
		bad := ch.Validate(map[string]any{"webhook": "ftp://x"})
		if bad == nil {
			t.Fatalf("%s: ftp 地址必须校验失败", kind)
		}
		assertChinese(t, kind+" bad url", bad.Error())
		if !strings.Contains(bad.Error(), "Webhook 地址无效") {
			t.Errorf("%s: 非法地址错误必须以 'Webhook 地址无效' 开头，实际得到 %q", kind, bad)
		}
	}
}

// TestChinaPlatformSendErrorsLocalized 检查钉钉·飞书·企业微信的发送失败原因以简体中文
// 写入投递历史(last_error)。平台名与 UI 渠道标签(DingTalk/Feishu/WeCom)保持一致。
// 服务端 errmsg 是 ASCII，所以错误里一出现谚文就说明骨架文案回退了。
func TestChinaPlatformSendErrorsLocalized(t *testing.T) {
	// 业务错误码（HTTP 200 + body 里 errcode/code != 0）路径。
	bizCases := []struct {
		kind     string
		resp     string
		wantSubs []string
	}{
		{KindDingTalk, `{"errcode":310000,"errmsg":"keyword not matched"}`, []string{"钉钉返回错误", "310000"}},
		{KindFeishu, `{"code":19021,"msg":"sign error"}`, []string{"飞书返回错误", "19021"}},
		{KindWeCom, `{"errcode":45009,"errmsg":"freq out of limit"}`, []string{"企业微信限流", "45009"}},
		{KindWeCom, `{"errcode":93000,"errmsg":"invalid webhook"}`, []string{"企业微信返回错误", "93000"}},
	}
	for _, tc := range bizCases {
		srv := capturePost(t, tc.resp, nil)
		ch, ok := Get(tc.kind)
		if !ok {
			t.Fatalf("%s 渠道未注册", tc.kind)
		}
		_, err := ch.Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
		if err == nil {
			t.Fatalf("%s: %s 响应必须报错", tc.kind, tc.resp)
		}
		if hasHangul(err.Error()) {
			t.Errorf("%s: 发送错误仍残留谚文: %q", tc.kind, err)
		}
		for _, sub := range tc.wantSubs {
			if !strings.Contains(err.Error(), sub) {
				t.Errorf("%s: 发送错误必须包含 %q，实际得到 %q", tc.kind, sub, err)
			}
		}
	}

	// 响应不是 JSON 时的解析失败路径："解析<平台>响应失败"。
	parseCases := []struct {
		kind     string
		platform string
		want     string
	}{
		{KindDingTalk, "DingTalk", "解析钉钉响应失败"},
		{KindFeishu, "Feishu", "解析飞书响应失败"},
		{KindWeCom, "WeCom", "解析企业微信响应失败"},
	}
	for _, pc := range parseCases {
		srv := capturePost(t, `not-json`, nil)
		ch, ok := Get(pc.kind)
		if !ok {
			t.Fatalf("%s 渠道未注册", pc.kind)
		}
		_, err := ch.Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
		if err == nil {
			t.Fatalf("%s: 非 JSON 响应必须报错", pc.kind)
		}
		if !strings.Contains(err.Error(), pc.want) {
			t.Errorf("%s: 解析失败错误必须包含 %q，实际得到 %q", pc.kind, pc.want, err)
		}
	}
}
