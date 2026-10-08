package agent

import (
	"strings"
	"testing"
	"unicode"
)

// A2: wrap-up / settlement 提示词中文化。
//
// 这些常量在 run 或 task 触及步数/时间预算而结束时，于 settlement 阶段注入，
// 直接指示最终总结并原样展示给用户。因此它们 (1) 必须用简体中文撰写，
// (2) 不得残留谚文，(3) 必须保留工具名与「一句话纯文本」这类指示语义。
//
// DB 种子把 wrapup_prompt / task_timeout_wrapup_prompt 留空（db/db.go 的内置
// agent INSERT 不填这两列），为空时回落到这些常量。也就是说这些常量是
// wrap-up 文案的唯一来源。
func TestWrapupPromptsLocalizedToChinese(t *testing.T) {
	all := map[string]string{
		"settleWrapUpPrompt":        settleWrapUpPrompt,
		"plannerWrapUpDefault":      plannerWrapUpDefault,
		"mainAgentWrapUpDefault":    mainAgentWrapUpDefault,
		"genericWrapUpDefault":      genericWrapUpDefault,
		"workerTaskTimeoutDefault":  workerTaskTimeoutDefault,
		"plannerTaskTimeoutDefault": plannerTaskTimeoutDefault,
	}

	hasScript := func(s string, table *unicode.RangeTable) bool {
		for _, r := range s {
			if unicode.Is(table, r) {
				return true
			}
		}
		return false
	}

	for name, p := range all {
		if !hasScript(p, unicode.Han) {
			t.Errorf("%s: 一个汉字都没有，未中文化", name)
		}
		// 工具名是 ASCII、谚文属 Hangul 区段，所以本地化完成后不应出现任何谚文。
		if hasScript(p, unicode.Hangul) {
			t.Errorf("%s: 残留谚文，翻译未完成: %q", name, p)
		}
	}

	// 工具名是标识符，必须原样保留、不翻译。
	// worker 系列（per-run 与 task-timeout）用 record_fact 写结论、report_finding
	// 写漏洞，并在最后要求用一句话纯文本做总结，这些指示必须保留。
	mustContain := func(name, p string, subs ...string) {
		for _, s := range subs {
			if !strings.Contains(p, s) {
				t.Errorf("%s: 指示语义 %q 必须保留，但缺失", name, s)
			}
		}
	}
	mustContain("settleWrapUpPrompt", settleWrapUpPrompt,
		"insert_assets", "record_fact", "report_finding", "一句话", "纯文本")
	mustContain("workerTaskTimeoutDefault", workerTaskTimeoutDefault,
		"insert_assets", "record_fact", "report_finding", "一句话", "纯文本")
	mustContain("genericWrapUpDefault", genericWrapUpDefault, "一句话", "纯文本")
	mustContain("mainAgentWrapUpDefault", mainAgentWrapUpDefault, "一句话", "纯文本")
	// planner 不产出总结句（只做判定后结束），并保留意图、目标、待办工具。
	mustContain("plannerWrapUpDefault", plannerWrapUpDefault, "add_intent", "prove_goal", "TodoWrite")
	mustContain("plannerTaskTimeoutDefault", plannerTaskTimeoutDefault, "prove_goal")
}

// per-run 与 task-timeout 的语义必须不同（planner 尤其如此）：per-run 表示
// 「只结束这一轮」，task-timeout 表示「整个任务结束」。常量映射被改乱就会串味。
func TestWrapupDefaultsRouting(t *testing.T) {
	if WrapupDefault("worker") != settleWrapUpPrompt {
		t.Error("worker 的 per-run 默认值不是 settleWrapUpPrompt")
	}
	if WrapupDefault("planner") != plannerWrapUpDefault {
		t.Error("planner 的 per-run 默认值不是 plannerWrapUpDefault")
	}
	if WrapupDefault("mainagent") != mainAgentWrapUpDefault {
		t.Error("mainagent 的 per-run 默认值不是 mainAgentWrapUpDefault")
	}
	// 未注册的键（自定义 agent）落到 generic。
	if WrapupDefault("unknown-agent") != genericWrapUpDefault {
		t.Error("未注册的键没有落到 genericWrapUpDefault")
	}
	// task-timeout 只有 worker/planner 有，其余返回空串（调用方回落到 per-run）。
	if TaskTimeoutWrapupDefault("worker") != workerTaskTimeoutDefault {
		t.Error("worker 的 task-timeout 默认值不是 workerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("planner") != plannerTaskTimeoutDefault {
		t.Error("planner 的 task-timeout 默认值不是 plannerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("mainagent") != "" {
		t.Error("mainagent 不应有 task-timeout 文案（必须是空串）")
	}
	// per-run 与 task-timeout 文案相同就说明语义区分消失了。
	if workerTaskTimeoutDefault == settleWrapUpPrompt {
		t.Error("worker 的 per-run 与 task-timeout 文案相同")
	}
	if plannerTaskTimeoutDefault == plannerWrapUpDefault {
		t.Error("planner 的 per-run 与 task-timeout 文案相同")
	}
}
