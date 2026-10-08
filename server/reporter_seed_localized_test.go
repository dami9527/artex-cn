package server

import (
	"testing"
)

// TestReporterSeedLabelsLocalized 是守护种子 reporter 智能体展示专用标签
// (reporterAgentName·reporterAgentDescription)保持简体中文的回归防御测试。
// 这两个字符串只经 `agentDTO`(server_mgmt.go)渲染到 system/agents UI，既不进入
// 任何智能体的 system 提示词(renderSystem 只按 key→段[A]
// agent.ReporterDefaultPrompt 组装)，也不参与基于模型的智能体选择(reporter 触发器
// 基于 report_finding 工具调用 OnToolCall·确定性)，属于纯 UI 标签。因此本地化它
// 不会碰 BRIEF 边界 #1(大脑不翻译)。反之，reporter 的大脑正文
// (agent.ReporterDefaultPrompt)与触发器注入消息(reporterToolCallMessage)是模型
// 输入，保留中文原文，因此不是本测试的对象。[[F35]] [[G133]]
//
// 本仓库固定为中文，reporterAgentLabels() 已无 locale 分支，因此只断言种子的
// 名称·描述是中文且不含谚文 —— 缺汉字或混入谚文都算回归。
func TestReporterSeedLabelsLocalized(t *testing.T) {
	name, desc := reporterAgentLabels()
	if name != reporterAgentName || desc != reporterAgentDescription {
		t.Fatalf("reporterAgentLabels() = (%q, %q)，期望 (%q, %q)", name, desc, reporterAgentName, reporterAgentDescription)
	}
	assertChineseError(t, "reporter_name", name)
	assertChineseError(t, "reporter_description", desc)
}

// TestRetesterSeedLabelsLocalized 把同一契约应用到 retester(漏洞复测智能体)。
// 这个名称·描述同样只渲染在系统界面的纯 UI 标签(工具说明·参数说明·actool.Errorf
// 之类的大脑输入另行保留原文 —— 参见 finding_retests.go 顶部注释)。
func TestRetesterSeedLabelsLocalized(t *testing.T) {
	name, desc := retesterAgentLabels()
	if name != retesterAgentName || desc != retesterAgentDescription {
		t.Fatalf("retesterAgentLabels() = (%q, %q)，期望 (%q, %q)", name, desc, retesterAgentName, retesterAgentDescription)
	}
	assertChineseError(t, "retester_name", name)
	assertChineseError(t, "retester_description", desc)
}
