package server

import "testing"

// 回归防御测试：保证 manager.go 的 work 智能体数量设置校验错误文案为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)，
// SetWorkers(n<=0) 分支在触及 m.pg.SetSetting 之前返回，因此在无 DB
// 的本机可以用空 Manager 真正驱动。一旦有人把这个字面量
// 改回非中文文案，本测试即失败。
//
// 调用图判定(参见 errWorkersPositive 常量注释)：这条文案只经配置 PUT 处理器
// (server.go:3512 → writeErr 400) 暴露，属于用户专用，没有智能体工具(actool)
// 路径，不是大脑输入。同文件第 384 行的启动恢复错误属于运维启动
// 日志性质，超出 F3b 范围(保留原文)，不是本测试的对象。
func TestSetWorkersErrorLocalized(t *testing.T) {
	err := (&Manager{}).SetWorkers(0)
	if err == nil {
		t.Fatal("work 智能体数量为 0 必须被拒绝")
	}
	assertChineseError(t, "set_workers_nonpositive", err.Error())
}
