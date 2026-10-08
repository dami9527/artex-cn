package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防御测试：保证 intent_intervention.go 的 Worker 介入 API 错误响应为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)。

// TestIntentInterventionErrorConstantsLocalized 断言 11 个响应常量全部为简体中文。
// 生命周期·意图状态路径需要 s.engine·Store 配置，在无 DB 的本机跑不
// 完，因此这些文案直接断言常量本身(conversations.go 先例)。4 条输入校验路径由
// 下面的 HTTP 测试检查到响应正文。
func TestIntentInterventionErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"request_too_large", errIntentRequestTooLarge},
		{"message_empty", errIntentMessageEmpty},
		{"message_too_long", errIntentMessageTooLong},
		{"bad_request_id", errIntentBadRequestID},
		{"task_deleting", errIntentTaskDeleting},
		{"task_paused", errIntentTaskPaused},
		{"task_queued", errIntentTaskQueued},
		{"task_terminal", errIntentTaskTerminal},
		{"task_settling", errIntentTaskSettling},
		{"inherited_readonly", errIntentInheritedReadonly},
		{"not_paused", errIntentNotPaused},
	}
	for _, c := range cases {
		assertChineseError(t, c.name, c.msg)
	}
}

// TestSendWorkerMessageInputValidationLocalized 把 4 条输入校验路径跑到真实 HTTP 响应
// 正文。sendWorkerMessage 的这 4 条路径只看 s.m.Task(查表)与请求正文，
// 不经 s.engine·Store·DB，因此 tasks 表里放一个任务即可无 DB 跑完。
// 还确认常量确实进入响应。
func TestSendWorkerMessageInputValidationLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}}

	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/intents/1/message", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		req.SetPathValue("iid", "1")
		rec := httptest.NewRecorder()
		s.sendWorkerMessage(rec, req)
		return rec
	}
	errBody := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var resp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应 JSON 解析失败: %v (正文 %q)", err, rec.Body.String())
		}
		return resp.Error
	}

	cases := []struct {
		name string
		body string
		code int
		want string
	}{
		// 超过 64KB 上限的合法 JSON 正文。MaxBytesReader 会在读取途中返回超限错误。
		{"request_too_large", `{"message":"` + strings.Repeat("a", maxWorkerMessageBytes+1024) + `"}`, http.StatusRequestEntityTooLarge, errIntentRequestTooLarge},
		{"message_empty", `{"message":"  "}`, http.StatusBadRequest, errIntentMessageEmpty},
		{"message_too_long", `{"message":"` + strings.Repeat("字", 4001) + `"}`, http.StatusBadRequest, errIntentMessageTooLong},
		{"bad_request_id", `{"message":"你好","request_id":"含 空格"}`, http.StatusBadRequest, errIntentBadRequestID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := call(c.body)
			if rec.Code != c.code {
				t.Fatalf("状态码 = %d, 期望 = %d (正文 %q)", rec.Code, c.code, rec.Body.String())
			}
			if got := errBody(t, rec); got != c.want {
				t.Fatalf("响应文案 = %q, 期望 = %q", got, c.want)
			}
			assertChineseError(t, c.name, errBody(t, rec))
		})
	}
}
