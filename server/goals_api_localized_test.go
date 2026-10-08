package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防御测试：保证 goals_api.go 的目标管理 API 错误响应为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)，响应正文
// 提取复用 task_categories 测试的 decodeErrorField(同属 package server)。

// 提取复用 decodeErrorField。TestGoalErrorConstantsLocalized 断言 7 个响应常量
// 全部为简体中文。目标查询·保存(目标不存在·读取失败)要经 Store·DB 才能到达，
// 在无 DB 的本机跑不完，这些文案直接断言常量本身(intent_intervention 先例)。
// 输入校验·删除屏障路径由下面的 HTTP 测试检查到响应正文。
func TestGoalErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"task_deleting_add":    errGoalTaskDeletingAdd,
		"task_deleting_edit":   errGoalTaskDeletingEdit,
		"task_deleting_delete": errGoalTaskDeletingDelete,
		"text_empty":           errGoalTextEmpty,
		"not_found":            errGoalNotFound,
		"read_after_add":       errGoalReadAfterAdd,
		"read_after_edit":      errGoalReadAfterEdit,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestGoalHandlersResponsesLocalized 把不触碰 DB 就能跑完的处理器路径
// 跑到真实 HTTP 响应正文，确认常量确实进入响应。三个处理器都按
// s.m.Task(查表) → s.engine.beginTaskOperation(sync.Map 删除屏障) → 请求正文
// 校验的顺序，因此 tasks 表里放一个任务加一个空 Engine 就能无 Store·DB 运行。
func TestGoalHandlersResponsesLocalized(t *testing.T) {
	// 已竖起删除屏障的服务器: beginTaskOperation 返回 false，因此产出 409。
	deletingServer := func() *Server {
		s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
		s.engine.deleting.Store("t1", true)
		return s
	}
	// 无屏障的服务器: beginTaskOperation 返回 true，因此继续走到输入校验。
	liveServer := func() *Server {
		return &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
	}

	newReq := func(body, gid string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/goals", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		if gid != "" {
			req.SetPathValue("gid", gid)
		}
		return req
	}

	cases := []struct {
		name    string
		handler func(*Server) http.HandlerFunc
		server  func() *Server
		body    string
		gid     string
		code    int
		want    string
	}{
		{
			name:    "add/task-deleting",
			handler: func(s *Server) http.HandlerFunc { return s.addGoal },
			server:  deletingServer,
			body:    `{"text":"证明 SQL 注入"}`,
			code:    http.StatusConflict,
			want:    errGoalTaskDeletingAdd,
		},
		{
			name:    "add/text-empty",
			handler: func(s *Server) http.HandlerFunc { return s.addGoal },
			server:  liveServer,
			body:    `{"text":"   "}`,
			code:    http.StatusBadRequest,
			want:    errGoalTextEmpty,
		},
		{
			name:    "edit/task-deleting",
			handler: func(s *Server) http.HandlerFunc { return s.editGoal },
			server:  deletingServer,
			body:    `{"text":"修改后的目标"}`,
			gid:     "5",
			code:    http.StatusConflict,
			want:    errGoalTaskDeletingEdit,
		},
		{
			name:    "edit/text-empty",
			handler: func(s *Server) http.HandlerFunc { return s.editGoal },
			server:  liveServer,
			body:    `{"text":"  "}`,
			gid:     "5",
			code:    http.StatusBadRequest,
			want:    errGoalTextEmpty,
		},
		{
			name:    "delete/task-deleting",
			handler: func(s *Server) http.HandlerFunc { return s.deleteGoal },
			server:  deletingServer,
			body:    ``,
			gid:     "5",
			code:    http.StatusConflict,
			want:    errGoalTaskDeletingDelete,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := c.server()
			rec := httptest.NewRecorder()
			c.handler(s)(rec, newReq(c.body, c.gid))
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
