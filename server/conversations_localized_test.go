package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防御测试：保证 conversations.go 的会话(聊天) API 错误响应为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。

// TestConversationErrorConstantsLocalized 断言 12 个响应常量全部为简体中文。
// 处理器先经 s.pg(w)(DB)，在无 DB 的本机跑不到最后，因此直接断言
// 常量本身(auth.go 先例)。含 %d 的格式串用真实参数填充后检查最终文案。
func TestConversationErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"request_too_large", convErrRequestTooLarge},
		{"agent_key_empty", convErrAgentKeyEmpty},
		{"agent_key_too_long", fmt.Sprintf(convErrAgentKeyTooLong, maxConversationAgentKeyRunes)},
		{"agent_not_found", convErrAgentNotFound},
		{"llm_profile", convErrLLMProfile},
		{"title_too_long", fmt.Sprintf(convErrTitleTooLong, maxConversationTitleRunes)},
		{"title_or_pinned", convErrTitleOrPinned},
		{"title_empty", convErrTitleEmpty},
		{"bad_conv_id", convErrBadConvID},
		{"ids_count", fmt.Sprintf(convErrIDsCount, maxConversationDeleteBatch)},
		{"message_empty", convErrMessageEmpty},
		{"busy", convErrBusy},
	}
	for _, c := range cases {
		assertChineseError(t, c.name, c.msg)
	}
}

// TestDecodeConversationRequestTooLargeLocalized 用真实 HTTP 响应
// 正文检查请求正文超限路径。decodeConversationRequest 是不经 Server(DB) 的包级函数，
// 无 DB 也能运行，可确认常量确实进入响应(请求正文超限 →
// 413 + 中文文案)。
func TestDecodeConversationRequestTooLargeLocalized(t *testing.T) {
	// 超过 64KB 上限的合法 JSON 正文。MaxBytesReader 会在读取途中返回超限错误。
	body := `{"title":"` + strings.Repeat("a", maxConversationRequestBytes+1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(body))
	rec := httptest.NewRecorder()
	var dst struct {
		Title string `json:"title"`
	}
	if decodeConversationRequest(rec, req, &dst) {
		t.Fatal("正文超过上限，decodeConversationRequest 却返回了 true")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d, 期望 = %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v (正文=%q)", err, rec.Body.String())
	}
	if resp.Error != convErrRequestTooLarge {
		t.Fatalf("响应 error = %q, 期望 = %q", resp.Error, convErrRequestTooLarge)
	}
	assertChineseError(t, "decode.too_large.response", resp.Error)
}

// TestConversationDefaultTitlesLocalized 断言会话的两个默认标题常量(F8)为中文且
// 互不相同。convDefaultTitle 既是创建默认值，也是自动标题分支的哨兵，
// 若改回旧文案，用户会在会话列表·删除对话框中看到非中文文案。
func TestConversationDefaultTitlesLocalized(t *testing.T) {
	assertChineseError(t, "default_title", convDefaultTitle)
	assertChineseError(t, "attachment_title", convAttachmentTitle)
	if convDefaultTitle == convAttachmentTitle {
		t.Fatal("默认标题与附件默认标题不能相同")
	}
}

// TestIsDefaultConversationTitle 固定自动标题分支的判定。创建默认值
// (convDefaultTitle)与空标题属于自动标题对象，用户自取标题则不是。创建
// 默认值与哨兵是同一常量，此处防止两者不一致导致自动标题失效的回归。
func TestIsDefaultConversationTitle(t *testing.T) {
	if !isDefaultConversationTitle("") {
		t.Fatal("空标题应当属于自动标题对象")
	}
	if !isDefaultConversationTitle(convDefaultTitle) {
		t.Fatalf("创建默认值 %q 应当属于自动标题对象", convDefaultTitle)
	}
	if isDefaultConversationTitle("用户自取的标题") {
		t.Fatal("用户自取的标题不应属于自动标题对象")
	}
}

// TestConversationRetestReasonsLocalized 断言复测终止原因的
// 三个常量(F9)为简体中文。这些值存入 finding_retests.error 列，并经复测面板 item.error
// 暴露，若改回旧文案，用户会在面板上看到非中文原因。
func TestConversationRetestReasonsLocalized(t *testing.T) {
	assertChineseError(t, "retest_failed_to_start", convRetestFailedToStart)
	assertChineseError(t, "retest_status_read_failed", convRetestStatusReadFailed)
	assertChineseError(t, "retest_stopped_or_closed", convRetestStoppedOrClosed)
}

// TestTranscriptErrorSummaryLocalized 固定活动记录的错误包装器(F9)。聊天轮次
// (无标签)与任务主 Agent("主 Agent")两种调用都产出同一 "(…出错：…)" 形式，
// 内层 err 原文原样保留，且包装器中不得出现谚文。
func TestTranscriptErrorSummaryLocalized(t *testing.T) {
	cases := []struct {
		name  string
		label string
		errm  string
		want  string
	}{
		{"chat_turn", "", "connection reset", "（出错：connection reset）"},
		{"main_agent", "主 Agent", "connection reset", "（主 Agent 出错：connection reset）"},
	}
	for _, c := range cases {
		got := transcriptErrorSummary(c.label, c.errm)
		if got != c.want {
			t.Fatalf("%s: transcriptErrorSummary = %q, 期望 = %q", c.name, got, c.want)
		}
		if !strings.Contains(got, c.errm) {
			t.Fatalf("%s: err 原文没有被保留: %q", c.name, got)
		}
		// 中文文案使用全角括号与冒号，确认包装器已是全角形式(F1 方针)。
		if !strings.ContainsAny(got, "（）：") {
			t.Fatalf("%s: 缺少全角括号/冒号: %q", c.name, got)
		}
		// 包装器标签必须是中文且不含谚文(err 原文不在检查范围，
		// 按 ASCII 固定)。无标签的聊天轮次也包含中文 "出错"。
		wrapper := strings.ReplaceAll(got, c.errm, "")
		assertChineseError(t, c.name+".wrapper", wrapper)
	}
}
