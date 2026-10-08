package server

import "testing"

// TestSideQuestionAPIErrorsLocalized 是 F21 的守卫：侧问
// (侧问) message surfaced by the HTTP wrapper (writeErr responses) and by the
// 交互状态(e.Error)的每一条用户可见文案都必须是简体中文 —— 含汉字、不含谚文。
// These reach the chat 侧问 panel alongside the sidequestion package errors, so
// 两层一起本地化，避免面板出现多语言混排。复用
// assertChineseError (F3a)。
func TestSideQuestionAPIErrorsLocalized(t *testing.T) {
	for _, c := range []struct{ label, msg string }{
		{"ctx_not_saved", sideErrCtxNotSaved},
		{"model_config_changed", sideErrModelConfigChanged},
		{"service_unavailable", sideErrServiceUnavailable},
		{"task_archived", sideErrTaskArchived},
		{"worker_deleted", sideErrWorkerDeleted},
		{"bad_question", sideErrBadQuestion},
		{"task_archiving", sideErrTaskArchiving},
		{"request_id_reused", sideErrRequestIDReused},
		{"no_snapshot", sideErrNoSnapshot},
		{"concurrency_limit", sideErrConcurrencyLimit},
		{"answer_stopped", sideErrAnswerStopped},
		{"answer_timeout", sideErrAnswerTimeout},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
