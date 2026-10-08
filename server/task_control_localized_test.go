package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防御测试：保证 task_control.go 的任务·意图控制 API 错误响应为简体中文。
// 中文判定复用 F3a 的 assertChineseError(含汉字·无谚文)，响应正文提取复用
// task_categories 测试的 decodeErrorField(同属 package server)。

// TestTaskControlErrorConstantsLocalized 断言 10 个控制响应常量
// 全部为简体中文。意图控制路径(applyIntentControl)第一行就要经 t.Store.GetNode(DB)，
// 在无 DB 的本机跑不到最后，因此那 6 个直接断言常量本身
// (intent_intervention·goals_api 先例)。任务控制·批量路径由下面两个测试用真实
// 函数执行与 HTTP 响应正文确认。
func TestTaskControlErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"task_deleting":          errTaskCtrlDeleting,
		"task_terminal_pause":    errTaskCtrlTerminalPause,
		"task_already_paused":    errTaskCtrlAlreadyPaused,
		"batch_size":             errTaskCtrlBatchSizeFmt,
		"intent_inherited":       errIntentCtrlInheritedReadonly,
		"intent_only_running":    errIntentCtrlOnlyRunningPause,
		"intent_only_paused":     errIntentCtrlOnlyPausedResume,
		"intent_state_conflict":  errIntentCtrlStateConflictFmt,
		"intent_only_deletable":  errIntentCtrlOnlyDeletable,
		"intent_reason_required": errIntentCtrlReasonRequired,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestTaskControlApplyResponsesLocalized 真正执行不经 DB·引擎状态的 pause 控制
// 路径，确认常量确实进入返回的错误。三个分支都按
// s.m.Task(查表) → 删除屏障 → t.lifecycleSnapshot()(直接读内存
// 字段 Status/Paused) 的顺序，因此 tasks 表里放一个任务加一个空 Engine
// 就能无 Store·DB 运行。成功路径(真正暂停)要经 ApplyTaskPause(DB)，
// 这里不驱动 —— 错误分支都在那之前返回。
func TestTaskControlApplyResponsesLocalized(t *testing.T) {
	// 终态任务: isTerminalStatus(done) 分支。
	terminalServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "done"}
		return &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}, tk
	}
	// 已暂停的运行中任务: lifecycle.Paused 分支。
	pausedServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "running", Paused: true}
		return &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}, tk
	}
	// 已竖起删除屏障的任务: IsDeleting 分支。
	deletingServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "running"}
		s := &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}
		s.engine.deleting.Store("1", true)
		return s, tk
	}

	cases := []struct {
		name  string
		setup func() (*Server, *Task)
		want  string
	}{
		{"terminal", terminalServer, errTaskCtrlTerminalPause},
		{"already-paused", pausedServer, errTaskCtrlAlreadyPaused},
		{"deleting", deletingServer, errTaskCtrlDeleting},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, tk := c.setup()
			_, err := s.applyTaskControl(tk, "pause")
			if err == nil {
				t.Fatalf("期望返回错误，实际返回了 nil")
			}
			if err.Error() != c.want {
				t.Fatalf("返回错误 = %q, 期望 = %q", err.Error(), c.want)
			}
			assertChineseError(t, c.name, err.Error())
		})
	}
}

// TestControlTasksBatchResponsesLocalized 用真实
// HTTP 检查批量控制端点的响应正文。① 空 task_ids 在访问 s.m 之前就以 writeErr 返回
// 400 + 数量错误，因此 &Server{} 也能到达。② 批量 pause 一个终态任务时，
// 任务控制错误会原样进入 200 items[].error，确认 applyTaskControl 的错误
// 会一路传到批量 JSON。
func TestControlTasksBatchResponsesLocalized(t *testing.T) {
	t.Run("size-error", func(t *testing.T) {
		s := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/control",
			strings.NewReader(`{"task_ids":[],"action":"pause"}`))
		rec := httptest.NewRecorder()
		s.controlTasksBatch(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, 期望 400 (正文 %q)", rec.Code, rec.Body.String())
		}
		got := decodeErrorField(t, rec.Body.Bytes())
		want := fmt.Sprintf(errTaskCtrlBatchSizeFmt, maxBatchControlIDs)
		if got != want {
			t.Fatalf("响应文案 = %q, 期望 = %q", got, want)
		}
		if !strings.Contains(got, "100") {
			t.Fatalf("数量上限 100 没有体现在文案里: %q", got)
		}
		assertChineseError(t, "size-error", got)
	})

	t.Run("item-error-surfaces", func(t *testing.T) {
		tk := &Task{ID: "1", Status: "done"}
		s := &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/control",
			strings.NewReader(`{"task_ids":["1"],"action":"pause"}`))
		rec := httptest.NewRecorder()
		s.controlTasksBatch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, 期望 = 200 (正文 %q)", rec.Code, rec.Body.String())
		}
		var out struct {
			Items []struct {
				ID    string `json:"id"`
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("响应 JSON 解析失败: %v (正文 %q)", err, rec.Body.String())
		}
		if len(out.Items) != 1 {
			t.Fatalf("items 个数 = %d, 期望 = 1", len(out.Items))
		}
		if out.Items[0].OK || out.Items[0].Error != errTaskCtrlTerminalPause {
			t.Fatalf("items[0] = %+v, error 期望 = %q", out.Items[0], errTaskCtrlTerminalPause)
		}
		assertChineseError(t, "item-error", out.Items[0].Error)
	})
}

// TestIntentStateConflictWrapsSentinel 确认意图状态冲突文案既是中文，
// 又用 %w 原样包装 db.ErrIntentStateConflict，errors.Is 识别不被破坏。
// 枚举值 paused 必须原样保留。
func TestIntentStateConflictWrapsSentinel(t *testing.T) {
	err := fmt.Errorf(errIntentCtrlStateConflictFmt, db.ErrIntentStateConflict)
	if !errors.Is(err, db.ErrIntentStateConflict) {
		t.Fatalf("errors.Is(.., ErrIntentStateConflict) = false, %%w 包装被破坏: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Fatalf("枚举值 paused 没有保留在文案里: %q", err.Error())
	}
	assertChineseError(t, "state-conflict", err.Error())
}
