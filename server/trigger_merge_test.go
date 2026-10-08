package server

import (
	"strings"
	"testing"
)

// 每个事件都重复一遍长任务目标是主要的膨胀来源。这些测试固定
// 修复结果：任务上下文表头(描述 + 目标)每个任务只渲染一次，
// 无论合并了多少次同一任务的触发。

const longGoal = "拿到题目 f2-05 的受保护 flag 并通过 submit_flag 提交；本题密文已高度收敛，flag 只能由二进制内嵌数据派生……" // 代表那段几千字的继承事实

func sameTaskFires(n int) []triggeredRun {
	items := make([]triggeredRun, n)
	for i := range items {
		items[i] = triggeredRun{
			agentKey: "tec_benchmark", taskID: 72, taskDesc: "f2-05 逆向", taskGoal: longGoal,
			message: "【本次由工具调用触发】\n工具: submit_flag\n入参: {...}\n返回: {correct:false}", mergeable: true,
		}
	}
	return items
}

func TestMergeAllRunsWritesTaskGoalOnce(t *testing.T) {
	out := mergeAllRuns(sameTaskFires(39))
	if got := strings.Count(out.message, longGoal); got != 1 {
		t.Fatalf("same-task goal should appear exactly once in a merged-all run, got %d", got)
	}
	if strings.Count(out.message, "── 触发 ") < 1 || !strings.Contains(out.message, "触发 39") {
		t.Fatalf("all 39 event bodies should be present: %q", out.message)
	}
	// 合并运行会内联嵌入表头，因此 finalTriggerMessage 不得重复添加。
	if out.taskDesc != "" || out.taskGoal != "" {
		t.Fatalf("merged run must clear taskDesc/taskGoal to avoid a duplicate header")
	}
	if finalTriggerMessage(out) != out.message {
		t.Fatalf("finalTriggerMessage must not prepend another header for a merged run")
	}
}

func TestMergeAllRunsGroupsInterleavedTasks(t *testing.T) {
	// 两个任务的触发交错到达(A,B,A,B)时，每个任务的上下文
	// 仍必须只出现一次 —— 按任务分组，而不是逐事件重复。
	mk := func(id int64, goal string) triggeredRun {
		return triggeredRun{agentKey: "a", taskID: id, taskDesc: "d", taskGoal: goal, message: "body", mergeable: true}
	}
	out := mergeAllRuns([]triggeredRun{mk(1, "GOAL_A"), mk(2, "GOAL_B"), mk(1, "GOAL_A"), mk(2, "GOAL_B")})
	if got := strings.Count(out.message, "GOAL_A"); got != 1 {
		t.Fatalf("task #1 goal should appear once despite interleaving, got %d", got)
	}
	if got := strings.Count(out.message, "GOAL_B"); got != 1 {
		t.Fatalf("task #2 goal should appear once despite interleaving, got %d", got)
	}
	if !strings.Contains(out.message, "共 2 个任务") {
		t.Fatalf("header should report 2 tasks: %q", out.message)
	}
	if got := strings.Count(out.message, "── 触发 "); got != 4 {
		t.Fatalf("all 4 event bodies should be present, got %d", got)
	}
}

func TestMergeTriggeredRunsWritesTaskGoalOnce(t *testing.T) {
	out := mergeTriggeredRuns(sameTaskFires(5))
	if got := strings.Count(out.message, longGoal); got != 1 {
		t.Fatalf("same-task goal should appear exactly once in a by-task merge, got %d", got)
	}
}

func TestFinalTriggerMessageSingleFirePrependsHeaderOnce(t *testing.T) {
	item := sameTaskFires(1)[0]
	msg := finalTriggerMessage(item)
	if got := strings.Count(msg, longGoal); got != 1 {
		t.Fatalf("single fire should carry the task goal exactly once, got %d", got)
	}
	if !strings.HasPrefix(msg, "【任务 #72") {
		t.Fatalf("single fire should be prefixed with the task-context header: %q", msg)
	}
}

func TestTaskContextHeaderEmptyForIntervalFire(t *testing.T) {
	if h := taskContextHeader(0, "", ""); h != "" {
		t.Fatalf("interval/none trigger (no task) must produce no header, got %q", h)
	}
	// 定时触发的消息必须原样通过。
	item := triggeredRun{message: "定时触发正文"}
	if finalTriggerMessage(item) != "定时触发正文" {
		t.Fatalf("interval fire message must pass through unchanged")
	}
}

func TestTaskContextHeaderTruncatesLongGoal(t *testing.T) {
	huge := strings.Repeat("很", 5000)
	h := taskContextHeader(72, "d", huge)
	if len([]rune(h)) > 800 { // 200 desc + 500 goal + 截断标记/装饰，远小于 5000
		t.Fatalf("header should be bounded even for a huge goal, got %d runes", len([]rune(h)))
	}
}

// TestTriggerSynthesisChineseFramingPreserved 把 P3 触发器消息合成部产生的中文
// 框架显式固定为保留对象(智能体大脑输入)(F10/F19 边界判定)。这些文案
// 既是进入 finalTriggerMessage → runTriggeredRun → ca.Chat 的 user 消息的智能体输入，
// 又暴露在记录中，属于双重用途，按 BRIEF 边界 #1(大脑不翻译 —— 保留 benchmark 行为)
// 不得翻译。上面的合并/表头行为测试已经固定了 `【任务 #`·`── 触发 `·`共 N 个任务`，
// 但任务上下文表头的目标框架与 by-task 合并提示尚未固定，
// 这里补上(防止意外非中文化的反向回归守卫 · 与 conversations.go 的保留注释配套)。
func TestTriggerSynthesisChineseFramingPreserved(t *testing.T) {
	// 任务上下文表头的目标框架(（目标：…）)保留 —— 输入不含 CJK，只验证框架。
	if h := taskContextHeader(7, "d", "g"); !strings.Contains(h, "（目标：") {
		t.Fatalf("任务上下文表头的目标框架被改动了(禁止翻译大脑输入): %q", h)
	}
	// 保留 by-task 合并提示的框架。
	if out := mergeTriggeredRuns(sameTaskFires(3)); !strings.Contains(out.message, "【本会话合并了") {
		t.Fatalf("by-task 合并提示的框架被改动了(禁止翻译大脑输入): %q", out.message)
	}
}
