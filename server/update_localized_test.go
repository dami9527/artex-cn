package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// F3b(update.go): 守护自更新端点用户可见文案为简体中文的回归测试。
// 前端(system/settings 的 update-card)会原样渲染 progress.message·reason·writeErr 正文，
// 因此这些文案一旦改回非中文，更新页面就会重新弹出非中文的提示与进度消息。
// assertChineseError·decodeErrorField 辅助函数复用先行的 F3b 测试文件(同属 package server)。

func TestUpdateMessageConstantsLocalized(t *testing.T) {
	assertChineseError(t, "updateMsgPreparing", updateMsgPreparing)
	assertChineseError(t, "updateMsgFailed", updateMsgFailed)
	assertChineseError(t, "updateMsgStaged", updateMsgStaged)
	assertChineseError(t, "updateErrInProgress", updateErrInProgress)
	assertChineseError(t, "updateErrRollbackInProgress", updateErrRollbackInProgress)

	// 格式串常量先填充占位符再检查(%q/%s 被替换，且不含谚文)。
	notRelease := fmt.Sprintf(updateErrNotReleaseFmt, "v0.0.0-dev")
	assertChineseError(t, "updateErrNotReleaseFmt", notRelease)
	if !strings.Contains(notRelease, "v0.0.0-dev") {
		t.Fatalf("updateErrNotReleaseFmt: 版本占位符没有被替换: %q", notRelease)
	}
	latest := fmt.Sprintf(updateErrAlreadyLatestFmt, "v1.2.3")
	assertChineseError(t, "updateErrAlreadyLatestFmt", latest)
	if !strings.Contains(latest, "v1.2.3") {
		t.Fatalf("updateErrAlreadyLatestFmt: 版本占位符没有被替换: %q", latest)
	}
}

// 用真实代码路径确认 begin/finish 经 SSE 输出的进度消息取自中文常量。
// 为避免污染全局 updHub 使用本地实例(不需要 DB·网络)。
func TestUpdateProgressMessagesLocalized(t *testing.T) {
	h := &updateHub{subs: map[chan updateProgress]struct{}{}}

	if !h.begin("v1.2.3") {
		t.Fatal("首次调用 begin 必须返回 true")
	}
	cur, running := h.snapshot()
	if !running {
		t.Fatal("begin 之后必须处于 running")
	}
	if cur.Message != updateMsgPreparing {
		t.Fatalf("准备中消息不一致: %q", cur.Message)
	}
	if h.begin("v1.2.4") {
		t.Fatal("进行中时 begin 必须返回 false")
	}

	h.finish(errors.New("下载失败"))
	cur, running = h.snapshot()
	if running {
		t.Fatal("finish 之后必须解除 running")
	}
	if cur.Message != updateMsgFailed {
		t.Fatalf("失败消息不一致: %q", cur.Message)
	}

	h.finish(nil)
	cur, _ = h.snapshot()
	if cur.Message != updateMsgStaged {
		t.Fatalf("就绪消息不一致: %q", cur.Message)
	}
}

// 用真实 HTTP、不需要 DB 确认 updateRollback 的「进行中无法回滚」409 响应是中文。
func TestUpdateRollbackInProgressLocalized(t *testing.T) {
	updHub.mu.Lock()
	prev := updHub.running
	updHub.running = true
	updHub.mu.Unlock()
	defer func() {
		updHub.mu.Lock()
		updHub.running = prev
		updHub.mu.Unlock()
	}()

	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/update/rollback", nil)
	rec := httptest.NewRecorder()
	s.updateRollback(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("期望状态码 409，实际 %d (正文 %s)", rec.Code, rec.Body.Bytes())
	}
	msg := decodeErrorField(t, rec.Body.Bytes())
	assertChineseError(t, "updateRollback 进行中 409", msg)
	if msg != updateErrRollbackInProgress {
		t.Fatalf("拒绝回滚文案不一致: %q", msg)
	}
}
