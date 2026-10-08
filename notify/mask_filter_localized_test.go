package notify

import (
	"errors"
	"strings"
	"testing"
)

// F4 ⑥: 把 notify/mask.go·filter.go 里暴露给用户的文案固定住——保存或更新通知渠道
// 配置时，server/notify_api.go 会把这些校验错误原样作为 writeErr(400) 返回。
// channel.go 没有面向用户的文案（全是注释），不在范围内。
// hasHan / hasHangul / assertChinese 定义在 notify_localized_test.go。

// TestFilterValidateLocalized 检查 min_severity 填错时的校验错误。
// 该错误在保存与更新路径（server/notify_api.go:264·369）以 400 暴露。
func TestFilterValidateLocalized(t *testing.T) {
	err := Filter{MinSeverity: "hgih"}.Validate()
	if err == nil {
		t.Fatal("最低级别非法却没有报错")
	}
	assertChinese(t, "Filter.Validate", err.Error())
	// low/medium/high/critical 是配置枚举，必须原样保留。
	for _, tok := range []string{"low", "medium", "high", "critical"} {
		if !strings.Contains(err.Error(), tok) {
			t.Errorf("级别标记 %q 丢失了: %q", tok, err.Error())
		}
	}
	// 合法值与空值必须通过。
	if err := (Filter{MinSeverity: "high"}).Validate(); err != nil {
		t.Errorf("high 是合法值却报错: %v", err)
	}
	if err := (Filter{}).Validate(); err != nil {
		t.Errorf("空的最低级别是合法值却报错: %v", err)
	}
}

// TestPrepareConfigUpdateUnknownKindLocalized 检查用未注册渠道类型更新配置时的错误
// （server/notify_api.go:326 处以 400 返回）。
func TestPrepareConfigUpdateUnknownKindLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("definitely-not-a-channel", map[string]any{}, map[string]any{})
	if err == nil {
		t.Fatal("渠道类型未注册却没有报错")
	}
	assertChinese(t, "PrepareConfigUpdate unknown kind", err.Error())
}

// TestDestinationChangedErrorLocalized 通过真实路径触发
// ErrDestinationChangedWithoutCredentials：PATCH 只把 webhook 的目标地址(url)换成
// 新值，对凭据(headers)不做任何标注。服务端会把它作为 400 返回。
func TestDestinationChangedErrorLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("webhook",
		map[string]any{
			"url":     "https://old.example.com/hook",
			"headers": map[string]any{"Authorization": "Bearer real-token"},
		},
		map[string]any{"url": "https://attacker.example.com/hook"},
	)
	if err == nil {
		t.Fatal("改了目标地址却未标注凭据，居然没有报错")
	}
	var de *ErrDestinationChangedWithoutCredentials
	if !errors.As(err, &de) {
		t.Fatalf("错误类型不符合预期: %T", err)
	}
	assertChinese(t, "ErrDestinationChangedWithoutCredentials", err.Error())
	// 变更的目标键与缺失的凭据键名都属于配置字段名，必须原样出现在消息里。
	if !strings.Contains(err.Error(), "url") || !strings.Contains(err.Error(), "headers") {
		t.Errorf("字段键名缺失: %q", err.Error())
	}
}

// TestRejectMaskedInContainersLocalized 通过真实路径触发结构体内部掩码哨兵的拒绝：
// 在 headers（对象，同时是凭据字段）里夹带掩码哨兵值的 PATCH。
func TestRejectMaskedInContainersLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("webhook",
		map[string]any{},
		map[string]any{"headers": map[string]any{"Authorization": MaskedPrefix + ":…abc123"}},
	)
	if err == nil {
		t.Fatal("在结构体内部夹带掩码哨兵，居然没有报错")
	}
	assertChinese(t, "rejectMaskedInContainers", err.Error())
	// 哨兵字面量(__masked__)会写进消息，便于运维定位是哪个值有问题。
	if !strings.Contains(err.Error(), MaskedPrefix) {
		t.Errorf("消息里没有掩码哨兵: %q", err.Error())
	}
}
