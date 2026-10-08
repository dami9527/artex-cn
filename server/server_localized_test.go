package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防御测试：保证 server.go 的用户可见响应为简体中文(F3b-server.go +
// F34)。中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)，响应正文
// 提取复用 task_categories 测试的 decodeErrorField(同属 package server)。
// 提取复用 task_categories 测试的 decodeErrorField。范围是 ① F3b: writeErr 24 处 +
// validateTaskProfileIDs 经 writeErr 暴露的 fmt.Errorf 2 处，② F34: 不经 writeErr 的
// 用户可见 5 处(chatUnavailableReason 的 3 种 return + testLLM·Web 搜索探针的
// writeJSON error 2 种)，③ F34(d): 规则模式聊天 fallbackChat 的 3 种响应
// (确认 2 种 + 现状摘要)与中文触发别名(意图/提示)识别。传给智能体的提示词·
// 工具说明·seed 意图摘要·默认标题(未命名任务)·日志保留原文，不是本测试的对象。

// TestServerErrorConstantsLocalized 断言全部响应常量为简体中文。
// 含 %d 的格式串常量还额外用真实格式化结果检查，
// 确认代入替换值后中文不被破坏。
func TestServerErrorConstantsLocalized(t *testing.T) {
	plain := map[string]string{
		"intent_control":        errTaskDeletingIntentControl,
		"intent_rerun":          errTaskDeletingIntentRerun,
		"intent_not_rerunnable": errIntentNotRerunnable,
		"source_invalid":        errCreateTaskSourceInvalid,
		"intercept_rules":       errCreateTaskInterceptRules,
		"category_invalid":      errCreateTaskCategoryInvalid,
		"company_invalid":       errCreateTaskCompanyInvalid,
		"llm_profile_invalid":   errLLMProfileInvalid,
		"asset_store_disabled":  errAssetStoreDisabled,
		"asset_id_required":     errAssetIDRequired,
		"python_not_detected":   errPythonNotDetected,
		"notify_base_url":       errNotifyBaseURLScheme,
		"notify_digest":         errNotifyDigestRange,
		"new_session":           errTaskDeletingNewSession,
		"new_message":           errTaskDeletingNewMessage,
		"main_agent_busy":       errMainAgentBusy,
		// F34: 不经 writeErr 的用户可见响应
		"chat_no_profile":       errChatNoLLMProfile,
		"chat_no_active":        errChatNoActiveLLMProfile,
		"chat_not_ready":        errChatLLMNotReady,
		"llm_test_no_api_key":   errLLMTestNoAPIKey,
		"web_search_no_results": errWebSearchProbeNoResults,
		// F34(d): 规则模式聊天 fallbackChat 的两个确认响应(后面会拼上输入文本)
		"fallback_intent": fallbackIntentInjected,
		"fallback_hint":   fallbackHintRecorded,
	}
	for label, msg := range plain {
		assertChineseError(t, label, msg)
	}

	formatted := map[string]string{
		"source_limit":     fmt.Sprintf(errCreateTaskSourceLimit, 8),
		"source_not_found": fmt.Sprintf(errCreateTaskSourceNotFound, 999),
		"company_limit":    fmt.Sprintf(errCreateTaskCompanyLimit, 32),
		"llm_profile_404":  fmt.Sprintf(errLLMProfileNotFound, 7),
		// F34(d): 规则模式现状摘要(代入资产·待处理意图·已确认漏洞数量)
		"fallback_status": fmt.Sprintf(fallbackChatStatus, 3, 2, 1),
	}
	for label, msg := range formatted {
		assertChineseError(t, label, msg)
	}
}

// TestServerHandlersResponsesLocalized 把不触碰 DB 就能跑完的处理器路径
// 跑到真实 HTTP 响应正文，确认常量确实进入响应
// (把常量改回非中文文案本测试即失败 = 对抗性非空性)。需要 DB·引擎·资产库的路径
// (分类/企业无效·LLM 配置不存在·未检测到 Python·通知配置·主 Agent 占用)
// 由上面的常量断言固定。
func TestServerHandlersResponsesLocalized(t *testing.T) {
	// 已竖起删除屏障的服务器: beginTaskOperation / IsDeleting 判定为「删除中」。
	deletingServer := func() *Server {
		s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
		s.engine.deleting.Store("t1", true)
		return s
	}
	// 无屏障的服务器: 继续走到输入校验。
	liveServer := func() *Server {
		return &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
	}

	cases := []struct {
		name    string
		req     func() *http.Request
		handler func(*Server) http.HandlerFunc
		server  func() *Server
		code    int
		want    string
	}{
		{
			name: "controlIntent/task-deleting",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/intents/5/control", strings.NewReader(`{"action":"pause"}`))
				r.SetPathValue("id", "t1")
				r.SetPathValue("iid", "5")
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.controlIntent },
			server:  deletingServer,
			code:    http.StatusConflict,
			want:    errTaskDeletingIntentControl,
		},
		{
			name: "rerunIntent/task-deleting",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/intents/5/rerun", nil)
				r.SetPathValue("id", "t1")
				r.SetPathValue("iid", "5")
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.rerunIntent },
			server:  deletingServer,
			code:    http.StatusConflict,
			want:    errTaskDeletingIntentRerun,
		},
		{
			name: "rerunBlocked/task-deleting",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/intents/rerun-blocked", nil)
				r.SetPathValue("id", "t1")
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.rerunBlocked },
			server:  deletingServer,
			code:    http.StatusConflict,
			want:    errTaskDeletingIntentRerun,
		},
		{
			name: "createTask/source-limit",
			req: func() *http.Request {
				// MaxTaskSourceCount=8 → 9 个即超限(循环之前返回)。
				r := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(
					`{"goal":"目标","source_task_ids":["1","2","3","4","5","6","7","8","9"]}`))
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.createTask },
			server:  liveServer,
			code:    http.StatusBadRequest,
			want:    fmt.Sprintf(errCreateTaskSourceLimit, 8),
		},
		{
			name: "createTask/source-invalid",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(
					`{"goal":"目标","source_task_ids":["abc"]}`))
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.createTask },
			server:  liveServer,
			code:    http.StatusBadRequest,
			want:    errCreateTaskSourceInvalid,
		},
		{
			name: "createTask/source-not-found",
			req: func() *http.Request {
				// 999 不在 tasks 表里 → 找不到(不访问 DB，只查表)。
				r := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(
					`{"goal":"目标","source_task_ids":["999"]}`))
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.createTask },
			server:  liveServer,
			code:    http.StatusBadRequest,
			want:    fmt.Sprintf(errCreateTaskSourceNotFound, 999),
		},
		{
			name: "createTask/company-limit",
			req: func() *http.Request {
				// MaxTaskCompanyCount=32 → 33 个会在 NormalizeTaskCompanyIDs(纯函数)报错。
				ids := make([]string, 33)
				for i := range ids {
					ids[i] = fmt.Sprintf("%d", i+1)
				}
				body := `{"goal":"目标","company_ids":[` + strings.Join(ids, ",") + `]}`
				return httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body))
			},
			handler: func(s *Server) http.HandlerFunc { return s.createTask },
			server:  liveServer,
			code:    http.StatusBadRequest,
			want:    fmt.Sprintf(errCreateTaskCompanyLimit, 32),
		},
		{
			name: "createTask/llm-profile-invalid",
			req: func() *http.Request {
				// llm_profile_ids:[0] → validateTaskProfileIDs 在 loadProfileConfig(DB) 之前返回。
				r := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(
					`{"goal":"目标","llm_profile_ids":[0]}`))
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.createTask },
			server:  liveServer,
			code:    http.StatusBadRequest,
			want:    errLLMProfileInvalid,
		},
		{
			name: "taskCoverageGraph/asset-store-disabled",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/tasks/t1/coverage-graph", nil)
				r.SetPathValue("id", "t1")
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.taskCoverageGraph },
			server:  liveServer, // Manager.assets==nil → Assets()==nil → 503
			code:    http.StatusServiceUnavailable,
			want:    errAssetStoreDisabled,
		},
		{
			name: "taskAssetRefs/asset-id-required",
			req: func() *http.Request {
				// 没有 asset_id 查询参数 → ParseInt("")=0 → <=0 → 400(调用 Assets() 之前)。
				r := httptest.NewRequest(http.MethodGet, "/api/tasks/t1/asset-refs", nil)
				r.SetPathValue("id", "t1")
				return r
			},
			handler: func(s *Server) http.HandlerFunc { return s.taskAssetRefs },
			server:  liveServer,
			code:    http.StatusBadRequest,
			want:    errAssetIDRequired,
		},
		{
			// F34(b): profile_id 留空(=不查 DB)且 api_key 也留空时，会在 TestConnection
			// 之前以 writeJSON{ok:false,error} 返回(status 200)。liveServer 的
			// llmCfg.APIKey 也是 ""，没有全局兜底密钥。
			name: "testLLM/no-api-key",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/llm/test", strings.NewReader(`{}`))
			},
			handler: func(s *Server) http.HandlerFunc { return s.testLLM },
			server:  liveServer,
			code:    http.StatusOK,
			want:    errLLMTestNoAPIKey,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := c.server()
			rec := httptest.NewRecorder()
			c.handler(s)(rec, c.req())
			if rec.Code != c.code {
				t.Fatalf("状态码 = %d, 期望 = %d (正文 %q)", rec.Code, c.code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("响应文案 = %q, 期望 = %q", got, c.want)
			}
			assertChineseError(t, c.name, got)
		})
	}
}

// TestChatUnavailableReasonLocalized 确认 F34(a) 的 chatUnavailableReason()
// 真正返回给用户的原因连到中文常量。pg 为 nil 时(测试中
// 没有 DB)，会直接落到「未就绪」路径，因此断言那个 return 就是
// errChatLLMNotReady(把常量改回非中文文案即失败 = 对抗性非空性)。
// 另两个原因(无配置·无启用配置)需要 pg(DB)，由上面的常量断言固定。
func TestChatUnavailableReasonLocalized(t *testing.T) {
	s := &Server{m: &Manager{}} // m.pg == nil → 走第三种 return 路径
	got := s.chatUnavailableReason()
	if got != errChatLLMNotReady {
		t.Fatalf("chatUnavailableReason() = %q, 期望 = %q", got, errChatLLMNotReady)
	}
	assertChineseError(t, "chat_not_ready", got)
}

// TestFallbackCommandParsing 确认规则模式聊天的触发关键字解析(fallbackCommand)
// 能识别中文别名(意图/提示)与英文触发词，并只剥掉关键字返回
// 剩下的参数。这是不触碰 Store·DB 的纯输入解析，无 DB 也能跑。
// 删掉中文别名用例 intent-cn 会失败，删掉英文别名用例 english-* 也会失败(非空性)。
func TestFallbackCommandParsing(t *testing.T) {
	cases := []struct {
		name, in, cmd, text string
	}{
		{"intent-cn", "意图 SQLi 注入点先确认", "intent", "SQLi 注入点先确认"},
		{"hint-cn", "提示 先看登录表单", "hint", "先看登录表单"},
		{"intent-cn-alt", "意图 扫描端口", "intent", "扫描端口"},
		{"hint-cn-alt", "提示 看登录", "hint", "看登录"},
		{"english-intent", "intent scan ports", "intent", "scan ports"},
		{"english-hint", "hint try admin:admin", "hint", "try admin:admin"},
		{"status-cn", "现在情况如何", "", "现在情况如何"},
		{"status-empty", "   ", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd, text := fallbackCommand(c.in)
			if cmd != c.cmd || text != c.text {
				t.Fatalf("fallbackCommand(%q) = (%q, %q), 期望 = (%q, %q)", c.in, cmd, text, c.cmd, c.text)
			}
		})
	}
}
