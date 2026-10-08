package server

import "testing"

// 回归防御测试：保证 goals.go 的任务恢复(admitPausedTask)前置条件校验错误文案
// 为简体中文。中文判定复用 F3a 的 assertChineseError(含汉字·
// 无谚文)。返回那两条文案的 admitTaskWhen(requirePaused=true) 路径要经
// s.concMu 锁之后的 s.m.Task()·s.engine.IsDeleting()·beginTaskOperation() 引擎关卡，
// 在没有 DB·引擎的本机跑不完，因此直接固定常量本身
// (与 finding_retests·finding_traffic 关卡后的路径同样处理)。一旦有人把这个字面量
// 改回非中文文案，本测试即失败。
//
// 调用图判定(参见 goals.go 常量块注释)：这两条文案只经单条(server.go:1194 →
// writeErr 409)·批量(task_control.go:298 → items[].error) 任务控制响应暴露，
// 属于用户专用。编排器路径(orchestration.go:343)固定 action="pause"，
// 不会触及恢复校验，因此不是大脑输入。
func TestGoalResumeErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"resume_terminal", errGoalResumeTerminal},
		{"resume_not_paused", errGoalResumeNotPaused},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
