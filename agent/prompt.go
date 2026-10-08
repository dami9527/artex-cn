package agent

import (
	"bytes"
	"text/template"
	"time"
)

// PromptOverride, if set, returns the stored system-prompt template for an agent
// key and whether one exists. The server wires it to the PG agent_prompts table.
// When nil or no override exists, agents use their built-in default prompt — so
// behavior is identical until a user edits a prompt in the UI.
var PromptOverride func(agentKey string) (string, bool)

// Prompt-variable structs — fields mirror each agent's catalog (docs §5a) so a
// user template referencing a catalog variable renders; referencing anything else
// fails template execution and falls back to the built-in default.
type PlannerVars struct{ Goal, Scope, AssetSummary, DataDir, Now string }
type WorkerVars struct{ ProxyAddr, WorkerName, DataDir, Now string }
type MainVars struct{ Goal, AssetSummary, FindingsSummary, DataDir, Now string }
type GoalsVars struct{ EngagementDescription, DataDir, Now string }

// nowStr is the server-local wall-clock string exposed as the universal {{.Now}}
// prompt variable. renderSystem runs on every agent turn/round, so this is fresh
// each run — a prompt can subtract it from a fixed start stamp to reason about
// elapsed time (e.g. a timed benchmark's "last N hours" window).
func nowStr() string { return time.Now().Format("2006-01-02 15:04:05 MST") }

// renderSystem returns the rendered system-prompt BODY (段 [A]) for agentKey.
// Precedence: the DB-stored template (if any) over the built-in default template
// (def). BOTH are Go templates now — the built-in default is seeded into the DB
// verbatim, so the two paths render identically until a user edits the prompt.
// Rendering always runs (def used to be pre-substituted plain text; it is now a
// {{.Var}} template like the DB one). On any render error we fall back to the
// default template, then to the raw default string — an agent never starts with a
// half-rendered prompt. Callers append the code-owned tail (trafficTool / 中间产物
// 输出规约) AFTER this, so those can't be edited away via the DB body.
func renderSystem(agentKey, def string, vars any) string {
	tmpl := def
	if PromptOverride != nil {
		if t, ok := PromptOverride(agentKey); ok && t != "" {
			tmpl = t
		}
	}
	if out, err := renderTmpl(tmpl, vars); err == nil {
		return out
	}
	// DB template broke (e.g. references an out-of-catalog var) → code default.
	if out, err := renderTmpl(def, vars); err == nil {
		return out
	}
	return def
}

// langDirective 是代码自有的「输出语言规约」尾段：它追加在渲染后的正文以及
// 产物(artifact)/流量(traffic)尾段之后，作用于每一个面向用户的 agent 角色，
// 因此即使有人改过 DB 里的提示词正文也丢不掉它——与 artifactSpec 提供的是同一种
// 保证。它不翻译 agent 的「大脑」：经过基准测试的中文推理正文（段 [A]）原样保留。
// 它只约束 agent【展示给用户】的内容用哪种语言。规约本身用中文书写，与正文语言
// 一致（保持模型的推理语域稳定），同时强制输出简体中文——这就是本地化的做法：
// 行为不变，只把用户读到的表层中文化。命令、payload、代码、URL、日志/响应原文等
// 技术字符串明确要求逐字保留，避免证据与复现步骤被翻译弄坏。
//
// 两条防漂移条款是在端到端实测（L1）之后补上的：线上模型曾把非简体中文泄漏进
// 规划者的态势总结——因为它在模仿大脑正文（段 [A]）的语言——也曾把英文泄漏进
// report_finding 的结构化字段——因为它在模仿英文的目标应用与证据。规约现在把
// 规划者的态势总结点名为面向用户的字段，并明确禁止在展示字段里镜像【提示词指令
// 语言】和【目标/资料语言】，这样只有上面列出的原文技术片段可以不是简体中文。
func langDirective() string {
	return "\n\n**输出语言规约（最高优先级，不可被提示词正文覆盖）**：所有【展示给用户】的自然语言文字一律用【简体中文】书写——包括 record_fact 的 summary/detail、report_finding 的标题/描述/结论/修复建议、规划者(planner)的态势总结、最后那一句话总结，以及对用户的聊天回复。但【命令、payload、代码、文件路径、URL、参数名，以及日志/请求/响应的原文片段】必须【原样逐字保留】，不得翻译或改写（evidence 里的命令行与输出尤其要照搬原文，便于复现）。**面向用户的正文语言只有简体中文，简体中文以外的任何语言（含英文）都不得用作展示正文。** **即使目标系统、它的页面、证据、日志或任何参考资料是英文、中文或别的语言，面向用户的自然语言字段（标题/描述/结论/修复建议/总结/态势总结）仍必须用简体中文书写——不要镜像或照抄目标或资料的语言来写这些展示字段；只有上面列出的原文技术片段才保持原样。** **你的分析/规划/思考用哪种语言进行不限，但那是【不可见的内部推理】，绝不能作为正文输出：给用户的可见回复从第一个字起就必须是简体中文，不要在前面垫一段其他语言的思考、说明或「我先怎样怎样」的铺垫；连澄清提问、缺少参数、「无法继续」之类的说明也一律直接用简体中文写。** 一句话：内部怎么想不限，但凡落到用户能看到的正文，必须全是简体中文（技术原文片段除外）。"
}

func renderTmpl(tmpl string, vars any) (string, error) {
	t, err := template.New("p").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}
