package server

import (
	"fmt"
	"strings"
	"testing"
)

// 回归防御测试：保证 engine.go·engine_timeout.go·goals.go 里「展示专用」的
// 运行时活动摘要常量为简体中文。这些摘要会出现在仪表盘记录(activity transcript)里，
// 但不带 node_id(任务级)，不会被任何智能体上下文读回 —— 回喂
// 路径(planner.workerOutput·get_worker_output 工具)只在 intent 范围(node_id)内挑
// 'result'/'text' 活动。因此本地化不会碰大脑输入(BRIEF 边界 #1)。
// 反之，会回喂给大脑的摘要(seed 意图摘要 `完成任务目标…`·默认说明 `未命名任务`
// 等)保留原文，不是本测试的对象。中文判定复用 assertChineseError(含
// 汉字·无谚文，intercept_archive_localized_test.go)。[[G132]]
func TestRuntimeActivitySummariesLocalized(t *testing.T) {
	// 没有格式参数的固定摘要: 应当保持中文。
	plain := map[string]string{
		"goalless_task_done": goallessTaskDoneSummary,
		"goal_breakdown_r0":  goalBreakdownRound0Summary,
		"queued_no_llm":      queuedNoLLMSummary,
		"queued_fifo":        queuedFIFOSummary,
	}
	for label, msg := range plain {
		assertChineseError(t, label, msg)
	}

	// %d 格式串: 格式参数仍然存在，格式化结果也是中文，且数字确实
	// 被代入。
	fmtCases := map[string]string{
		"planner_round":       plannerRoundSummaryFmt,
		"timeout_final_round": timeoutFinalRoundSummaryFmt,
		"queued_conc_limit":   queuedConcurrencyLimitSummaryFmt,
	}
	for label, f := range fmtCases {
		if !strings.Contains(f, "%d") {
			t.Fatalf("%s: 格式参数 %%d 不见了: %q", label, f)
		}
		got := fmt.Sprintf(f, 7)
		assertChineseError(t, label, got)
		if !strings.Contains(got, "7") {
			t.Fatalf("%s: 格式参数没有体现在结果里: %q", label, got)
		}
	}
}
