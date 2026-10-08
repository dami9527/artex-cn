package server

import "testing"

// 回归防御测试：保证 finding_retests.go 的用户可见 HTTP 错误响应文案为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)，
// 返回那四条文案的 startFindingRetest 处理器一上来就经 s.pg(w)(DB)
// 关卡，在无 DB 的本机跑不完，因此直接固定常量本身
// (与 finding_traffic·goals_api·notify_api 的 DB 关卡路径同样处理)。一旦有人把
// 这个字面量改回非中文文案，本测试即失败。
//
// 工具说明(get_finding_retest_context·record_finding_retest_result)·参数说明·
// actool.Errorf(轮次未关联提示)以及 seedFindingRetester 的 DB 种子智能体名称·
// 配置·note 属于智能体读取的大脑输入或种子，刻意保留原文(中文)，
// 不是本测试的断言对象(参见 finding_retests.go 常量块注释)。
func TestFindingRetestErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"notes_too_long", errFindingRetestNotesTooLong},
		{"agent_missing", errFindingRetestAgentMissing},
		{"tool_required", errFindingRetestToolRequired},
		{"service_stopping", errFindingRetestServiceStopping},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
