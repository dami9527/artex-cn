package intercept

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// hasHangul 判断 s 是否含有谚文音节。
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

// hasHan 判断 s 是否含有 CJK 汉字。
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// assertChinese 在 s 缺汉字或残留谚文时失败。任何本地化文案一旦回退成谚文就会
// 触发 hasHangul，所以这条断言不是空转。
func assertChinese(t *testing.T, label, s string) {
	t.Helper()
	if !hasHan(s) {
		t.Errorf("%s: 没有汉字: %q", label, s)
	}
	if hasHangul(s) {
		t.Errorf("%s: 仍残留谚文: %q", label, s)
	}
}

// TestInterceptMessagesLocalized 钉住面向用户的判定与审批消息为简体中文。
// 它们出现在审批记录原因、活动流，以及请求已处理时的 409 响应里。
func TestInterceptMessagesLocalized(t *testing.T) {
	for _, c := range []struct{ name, s string }{
		{"msgReviewContextIncomplete", msgReviewContextIncomplete},
		{"msgModelApprovalFailed", msgModelApprovalFailed},
		{"msgModelOutputUnparsable", msgModelOutputUnparsable},
		{"reasonWorkCanceled", reasonWorkCanceled},
		{"reasonWorkCanceledPreExec", reasonWorkCanceledPreExec},
		{"reasonApprovalTimeout", reasonApprovalTimeout},
		{"reasonManualDeny", reasonManualDeny},
		{"reasonManualAllow", reasonManualAllow},
		{"ErrAlreadyDecided", ErrAlreadyDecided.Error()},
	} {
		assertChinese(t, c.name, c.s)
	}
}

// TestJudgeActionLabelLocalized 检查三种判定结论标签是简体中文
// （放行 / 拦截 / 转人工审批），且未知动作仍然原样透传。
func TestJudgeActionLabelLocalized(t *testing.T) {
	for _, action := range []string{"allow", "deny", "ask"} {
		assertChinese(t, "judgeActionLabel("+action+")", judgeActionLabel(action))
	}
	if got := judgeActionLabel("weird"); got != "weird" {
		t.Errorf("judgeActionLabel(weird) = %q, 期望透传原值", got)
	}
}

// TestDefaultMessageLocalized 检查由规则生成的拒绝/审批消息是简体中文，
// 且仍然内嵌规则名；allow 保持为空。
func TestDefaultMessageLocalized(t *testing.T) {
	for _, action := range []string{"deny", "ask"} {
		msg := defaultMessage(action, "R1")
		assertChinese(t, "defaultMessage("+action+")", msg)
		if !strings.Contains(msg, "R1") {
			t.Errorf("defaultMessage(%s) 丢失了规则名: %q", action, msg)
		}
	}
	if got := defaultMessage("allow", "R1"); got != "" {
		t.Errorf("defaultMessage(allow) = %q, 期望为空", got)
	}
}

// TestToolApprovalSummaryLocalized 检查活动摘要为简体中文，且仍能被 transcript.tsx
// 解析——它用 /\(#(\d+)\)/ 提取待处理 id，用 /工具\s+(\S+)\s+请求审批/ 提取工具名。
func TestToolApprovalSummaryLocalized(t *testing.T) {
	s := fmt.Sprintf(msgToolApprovalRequestFmt, "Bash", 42)
	assertChinese(t, "msgToolApprovalRequestFmt", s)
	if !strings.Contains(s, "(#42)") {
		t.Errorf("摘要丢失了 (#N) 标记（transcript.tsx 的 pending_id 正则）: %q", s)
	}
	if !strings.Contains(s, "工具 Bash 请求审批") {
		t.Errorf("摘要丢失了 '工具 X 请求审批' 形态（transcript.tsx 的 toolName 正则）: %q", s)
	}
}
