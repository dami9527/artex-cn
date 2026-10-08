package server

import "testing"

// 回归防御测试：保证 finding_traffic.go 的用户可见 HTTP 错误响应文案为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)，
// 返回那四条文案的路径(findingTrafficAccess·bindFindingTraffic·
// editFindingTraffic)全都在 s.m.pg.GetFinding(DB) 关卡之后，在无 DB 的本机
// 跑不完，因此直接固定常量本身(与 goals_api·notify_api 的 DB 关卡
// 路径同样处理)。一旦有人把这个字面量改回非中文文案，本测试即失败。
//
// readEvidencePreview 的 errors.New(offset/length 校验)·二进制正文占位符，
// 以及 roTool("get_finding_traffic") 工具说明会作为智能体工具结果回喂到大脑，
// 属于大脑输入，刻意保留原文(中文)，不是本测试的断言对象。
func TestFindingTrafficErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"inherited_readonly", errFindingTrafficInheritedReadonly},
		{"select_required", errFindingTrafficSelectRequired},
		{"version_required", errFindingTrafficVersionRequired},
		{"binding_ids_required", errFindingTrafficBindingIDsRequired},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
