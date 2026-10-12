package agent

import (
	"strings"
	"testing"
)

func TestRenderSystemOverrideAndFallback(t *testing.T) {
	t.Cleanup(func() { PromptOverride = nil })

	// no override → built-in default
	PromptOverride = nil
	if got := renderSystem("planner", "DEFAULT", PlannerVars{Goal: "g"}); got != "DEFAULT" {
		t.Fatalf("no override should give default, got %q", got)
	}

	// override → rendered with vars
	PromptOverride = func(k string) (string, bool) {
		if k == "planner" {
			return "目标:{{.Goal}} 范围:{{.Scope}}", true
		}
		return "", false
	}
	if got := renderSystem("planner", "DEFAULT", PlannerVars{Goal: "拿下X", Scope: "*.x.com"}); got != "目标:拿下X 范围:*.x.com" {
		t.Fatalf("override render: %q", got)
	}

	// override referencing a non-catalog var → execution error → fallback to default
	PromptOverride = func(k string) (string, bool) { return "{{.NotInCatalog}}", true }
	if got := renderSystem("planner", "DEFAULT", PlannerVars{Goal: "x"}); got != "DEFAULT" {
		t.Fatalf("bad var should fall back to default, got %q", got)
	}

	// full plannerSystem path: DB body [A] is honored, then the code-owned tail
	// [C] (中间产物输出规约) is ALWAYS appended — editing the body can't drop it.
	PromptOverride = func(k string) (string, bool) { return "PLANNER {{.Goal}}", true }
	got := plannerSystem("拿下X", "/data", "/data")
	if !strings.HasPrefix(got, "PLANNER 拿下X") {
		t.Fatalf("plannerSystem body not honored: %q", got)
	}
	if !strings.Contains(got, "中间产物输出规约") || !strings.Contains(got, "/data") {
		t.Fatalf("plannerSystem missing code-owned artifact tail: %q", got)
	}

	// worker dual-text via {{if .ProxyAddr}} in a user template, plus the code tail:
	// [B] trafficTool present only when RECORDING (caCert set — the MITM is on, so
	// the traffic_* tools exist), [C] artifact spec always present. The trafficTool
	// block is gated on the CA (arg 2), NOT on ProxyAddr — a global egress proxy
	// with capture off routes traffic but records nothing.
	PromptOverride = func(k string) (string, bool) {
		return "{{if .ProxyAddr}}走代理 {{.ProxyAddr}}{{else}}手动{{end}}", true
	}
	recording := workerSystem("127.0.0.1:8080", "/ca.pem", "/data", "/data")
	if !strings.HasPrefix(recording, "走代理 127.0.0.1:8080") {
		t.Fatalf("worker proxy branch body: %q", recording)
	}
	if !strings.Contains(recording, "traffic_search") {
		t.Fatalf("worker while recording should inject trafficTool: %q", recording)
	}
	if strings.Contains(recording, "traffic_refs") {
		t.Fatalf("worker bypassed shared optional evidence policy: %q", recording)
	}
	if !strings.Contains(recording, "中间产物输出规约") {
		t.Fatalf("worker missing artifact tail: %q", recording)
	}
	// Egress proxy set but capture OFF (no CA): the ProxyAddr template branch still
	// renders, but the trafficTool block must NOT — those tools are not registered.
	egressOnly := workerSystem("127.0.0.1:8080", "", "/data", "/data")
	if !strings.HasPrefix(egressOnly, "走代理 127.0.0.1:8080") {
		t.Fatalf("worker egress-only branch body: %q", egressOnly)
	}
	if strings.Contains(egressOnly, "traffic_search") {
		t.Fatalf("worker without recording must NOT inject trafficTool: %q", egressOnly)
	}
	noProxy := workerSystem("", "", "/data", "/data")
	if !strings.HasPrefix(noProxy, "手动") {
		t.Fatalf("worker no-proxy branch body: %q", noProxy)
	}
	if strings.Contains(noProxy, "traffic_search") {
		t.Fatalf("worker without proxy must NOT inject trafficTool: %q", noProxy)
	}
}

// langMarker 是能证明 langDirective 选中了某种展示语言的无歧义子串：这些是该语言在
// （用中文写成的）语言规约里的自称写法，因此命中就说明规约点名了这种语言。
// 供下面的语言规约测试与角色追加测试使用。
var langMarker = map[string]string{
	"zh": "简体中文",
	"en": "English",
	"es": "Español",
}

// TestLangDirectiveLocalized 钉住三种展示语言下的输出语言规约尾巴：每种语言的规约都要
// 点名该语言、保留原文技术片段、禁止镜像目标或资料的语言，并点名规划者的态势总结。
// 对每种非中文展示语言，「绝不能把中文输出给用户」这一条都必须存在；展示语言本身是
// 中文时，中文输出正是目标，该条必须缺席。
func TestLangDirectiveLocalized(t *testing.T) {
	for _, code := range SupportedOutputLangs {
		dir := langDirective(code)
		if !strings.Contains(dir, langMarker[code]) {
			t.Fatalf("%s: langDirective 必须点名展示语言 (%q)，实际得到 %q", code, langMarker[code], dir)
		}
		if !strings.Contains(dir, "payload") || !strings.Contains(dir, "原样逐字保留") {
			t.Fatalf("%s: langDirective 必须逐字保留命令/payload，实际得到 %q", code, dir)
		}
		if !strings.Contains(dir, "不要镜像或照抄目标") {
			t.Fatalf("%s: langDirective 必须禁止镜像目标或资料的语言，实际得到 %q", code, dir)
		}
		if !strings.Contains(dir, "态势") {
			t.Fatalf("%s: langDirective 必须点名规划者态势总结属于面向用户的字段，实际得到 %q", code, dir)
		}
		// 「绝不能把中文输出给用户」这一条适用于每种非中文展示语言（防止规划者
		// 态势总结漂移）。展示语言本身就是中文时，中文输出正是目标，因此该条必须缺席。
		hasNoZhClause := strings.Contains(dir, "也绝不能把中文输出给用户")
		if code == "zh" && hasNoZhClause {
			t.Fatalf("zh: 展示语言是中文时，规约不得禁止输出中文，实际得到 %q", dir)
		}
		if code != "zh" && !hasNoZhClause {
			t.Fatalf("%s: langDirective 必须禁止把中文泄漏给用户，实际得到 %q", code, dir)
		}
	}
}

// TestLangDirectiveAppendedToUserFacingRoles 钉住语言规约这条尾巴的接线：每个面向用户的
// 角色，其 system prompt 末尾都必须带上由代码追加的、当前所选语言的输出语言规约，
// 数据库改动过的正文不可能把它丢掉。通过角色解析所用的 OutputLanguage 钩子，
// 对每种支持的展示语言各跑一遍。
func TestLangDirectiveAppendedToUserFacingRoles(t *testing.T) {
	t.Cleanup(func() { PromptOverride = nil; OutputLanguage = nil })

	// 即使数据库正文是纯非规约文本，代码追加的尾巴仍会挂在每个面向用户的构造函数
	// 后面——与中间产物尾巴是同一套保证。自定义正文永远无法把输出语言这条强制
	// 要求翻译掉。
	PromptOverride = func(string) (string, bool) { return "BODY-ONLY", true }

	for _, code := range SupportedOutputLangs {
		OutputLanguage = func() string { return code }
		marker := langMarker[code]
		cases := map[string]string{
			"worker":    workerSystem("", "", "/data", "/data"),
			"planner":   plannerSystem("g", "/data", "/data"),
			"mainagent": mainAgentSystem("g", "/data", "/data"),
			"chat":      chatSystem("chat", "/data", "/data"),
			// goals 同样面向用户：set_goals/set_constraints 持久化的目标与约束节点会
			// 展示在 UI 的图谱/计划页。withScope=true 会走更长的组装路径（正文 + 范围
			// 尾巴），所以语言尾巴仍然必须落在最后——在正文与代码拥有的范围尾巴之后。
			"goals": goalsSystem("/data", true),
		}
		for role, sys := range cases {
			if !strings.HasPrefix(sys, "BODY-ONLY") {
				t.Fatalf("%s/%s: 数据库正文未被采用: %q", code, role, sys)
			}
			if !strings.Contains(sys, marker) {
				t.Fatalf("%s/%s: 缺少输出语言尾巴 (%q): %q", code, role, marker, sys)
			}
			// 规约就是那条尾巴——必须排在正文之后（就近生效）。
			if strings.Index(sys, marker) <= strings.Index(sys, "BODY-ONLY") {
				t.Fatalf("%s/%s: langDirective 必须追加在正文之后: %q", code, role, sys)
			}
		}
	}
}
