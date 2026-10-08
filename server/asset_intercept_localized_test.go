package server

import "testing"

// 回归防御测试：保证 asset_intercept.go·task_intercept.go 的拦截规则校验器返回的
// 用户可见错误文案为简体中文。两个校验器
// (validateAssetInterceptRuleReq·validateTaskInterceptRuleReq) 是不使用 DB 的纯
// 函数，因此在固定常量之外，还直接调用真实错误路径检查返回文案。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。
// 字段名·枚举值(pattern·exact_ip·cidr·kind·action·block·allow)是线上标识符，保持 ASCII，
// 不会被汉字判定拦住。一旦有人把周围的说明改回非中文文案即失败。
func TestAssetInterceptErrorsLocalized(t *testing.T) {
	// 常量固定
	for _, c := range []struct{ label, msg string }{
		{"pattern_empty", errAssetInterceptPatternEmpty},
		{"bad_exact_ip", errAssetInterceptInvalidExactIPFmt},
		{"bad_cidr", errAssetInterceptInvalidCIDRFmt},
		{"bad_kind", errAssetInterceptInvalidKindFmt},
		{"bad_action", errTaskInterceptInvalidAction},
	} {
		assertChineseError(t, c.label, c.msg)
	}

	// 真实路径: validateAssetInterceptRuleReq 的四个拒绝分支
	for _, c := range []struct {
		label string
		req   assetInterceptRuleReq
	}{
		{"empty_pattern", assetInterceptRuleReq{Kind: "exact_ip", Pattern: "   "}},
		{"invalid_exact_ip", assetInterceptRuleReq{Kind: "exact_ip", Pattern: "not-an-ip"}},
		{"invalid_cidr", assetInterceptRuleReq{Kind: "cidr", Pattern: "nonsense"}},
		{"invalid_kind", assetInterceptRuleReq{Kind: "nope", Pattern: "example.com"}},
	} {
		req := c.req
		err := validateAssetInterceptRuleReq(&req)
		if err == nil {
			t.Fatalf("%s: 没有报错(校验通过了)", c.label)
		}
		assertChineseError(t, c.label, err.Error())
	}

	// 合法输入不会被拒绝
	okReq := assetInterceptRuleReq{Kind: "cidr", Pattern: "192.168.0.0/16"}
	if err := validateAssetInterceptRuleReq(&okReq); err != nil {
		t.Fatalf("合法的 CIDR 被拒绝了: %v", err)
	}

	// 真实路径: validateTaskInterceptRuleReq 的 action 拒绝分支
	badAction := taskInterceptRuleReq{Action: "nope", Kind: "exact_domain", Pattern: "example.com"}
	if err := validateTaskInterceptRuleReq(&badAction); err == nil {
		t.Fatalf("bad_action_path: 没有报错(校验通过了)")
	} else {
		assertChineseError(t, "bad_action_path", err.Error())
	}

	// action 留空会规范化为 block 默认值并通过
	defaultAction := taskInterceptRuleReq{Action: "", Kind: "exact_domain", Pattern: "example.com"}
	if err := validateTaskInterceptRuleReq(&defaultAction); err != nil {
		t.Fatalf("默认 action 规范化失败: %v", err)
	}
	if defaultAction.Action != "block" {
		t.Fatalf("默认 action 没有被规范化为 block: %q", defaultAction.Action)
	}
}
