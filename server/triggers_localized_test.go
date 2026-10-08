package server

import "testing"

// 回归防御测试：保证 triggers.go 的触发器配置校验错误文案为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。

// TestTriggerErrorConstantsLocalized 断言 3 个错误常量全部为简体中文。
func TestTriggerErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"no_condition", errTriggerNoCondition},
		{"tool_set_empty", errTriggerToolSetEmpty},
		{"custom_only", errTriggerCustomOnly},
	}
	for _, c := range cases {
		assertChineseError(t, c.name, c.msg)
	}
}

// TestValidateTriggerLocalized 真正调用 validateTrigger 纯函数，确认条件缺失·工具
// 未选择两条路径返回中文常量，而合法输入返回空字符串。这个返回值会被
// pgCreateTrigger·pgUpdateTrigger 以 writeErr(400, msg) 原样暴露给用户。
func TestValidateTriggerLocalized(t *testing.T) {
	if msg := validateTrigger(&triggerReq{}); msg != errTriggerNoCondition {
		t.Fatalf("条件缺失文案 = %q, 期望 = %q", msg, errTriggerNoCondition)
	}
	if msg := validateTrigger(&triggerReq{OnToolCall: true}); msg != errTriggerToolSetEmpty {
		t.Fatalf("工具未选择文案 = %q, 期望 = %q", msg, errTriggerToolSetEmpty)
	}
	if msg := validateTrigger(&triggerReq{OnFinding: true}); msg != "" {
		t.Fatalf("合法输入文案 = %q, 期望为空字符串", msg)
	}
	if msg := validateTrigger(&triggerReq{OnToolCall: true, ToolNames: []string{"nmap"}}); msg != "" {
		t.Fatalf("合法工具调用文案 = %q, 期望为空字符串", msg)
	}
}
