package server

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防御测试：保证 engine.go 的任务控制·意图介入错误中「用户可见」的
// 文案为简体中文。中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。
//
// 调用图判定(参见 engine.go 常量块注释)：
//   - ControlWork 的四个错误(无运行中的 work·已在进行控制·等待收尾被取消·等待收尾
//     超时) → applyIntentControl(task_control.go) → controlIntent(server.go:1236) →
//     writeErr 409。属于用户专用，不触及智能体工具(actool)路径。
//   - runDetachedIntent 的两个错误(Worker 未就绪·恢复 CAS 冲突) →
//     sendWorkerMessage(intent_intervention.go:147) → writeErr。属于用户专用。
//   - 反之 SteerWork·KillWork(steer_work·kill_work 工具的 actool.Errorf)·
//     transitionIntentState(内部状态迁移日志)的中文属于大脑输入·日志，予以保留，
//     不是本测试的对象。

// TestControlWorkNoRunningWorkErrorLocalized 在无运行中的 work 时直接驱动 ControlWork，
// 确认它真的返回中文错误，无需 DB·引擎。run==nil 分支只读 e.work 表，
// 因此用 NewEngine(nil) 就能跑到最后。它用 %w 包装 errWorkControlConflict 哨兵，
// 所以也确认 errors.Is 关系一并保留。一旦有人把这个字面量改回非中文文案即失败。
func TestControlWorkNoRunningWorkErrorLocalized(t *testing.T) {
	e := NewEngine(nil)
	err := e.ControlWork(context.Background(), 42, "pause")
	if err == nil {
		t.Fatal("没有运行中的 work，ControlWork 却返回了 nil")
	}
	if !errors.Is(err, errWorkControlConflict) {
		t.Fatalf("errors.Is(err, errWorkControlConflict) = false, err=%v", err)
	}
	assertChineseError(t, "control_work_no_running", err.Error())
}

// TestEngineUserFacingErrorsLocalized 固定其余被引擎/DB 关卡或协程时序挡在后面、
// 难以跑完的用户可见格式串(与 finding_retests·
// finding_traffic 关卡后的路径同样处理)。格式串用代表性参数填充后再判定。
func TestEngineUserFacingErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"work_control_busy", fmt.Errorf(errWorkControlBusyFmt, errWorkControlConflict, int64(42), "pause").Error()},
		{"work_control_wait", fmt.Errorf(errWorkControlWaitFmt, int64(42), "pause", context.DeadlineExceeded).Error()},
		{"detached_worker_not_ready", errDetachedWorkerNotReady},
		{"detached_state_conflict", fmt.Errorf(errIntentCtrlStateConflictFmt, db.ErrIntentStateConflict).Error()},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
