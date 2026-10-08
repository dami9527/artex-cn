package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 守护认证响应文案中文化(待办 F3b)的回归防御测试。即使没有 DB，
// 也用两条可跑通的路径检查: ① requireAuth 中间件不经 pg，因此检查真实 HTTP
// 响应正文；② 其余处理器文案是具名常量，直接检查常量本身。
// 中文判定复用 F3a 的 assertChineseError 辅助函数(含汉字·无谚文)。

// TestRequireAuthMessagesLocalized 用真实 HTTP 处理器确认: 没有 token 或 token
// 非法时 requireAuth 返回的 401 响应正文是中文且不含谚文。这个
// 中间件不使用数据库，因此在无 DB 的环境也能跑到最后。
func TestRequireAuthMessagesLocalized(t *testing.T) {
	s := &Server{jwtKey: []byte(strings.Repeat("k", 32))}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := s.requireAuth(next)

	cases := []struct {
		name   string
		bearer string
		want   string
	}{
		{"无 token", "", authErrUnauthorized},
		{"token 无效", "Bearer not-a-valid-token", authErrTokenInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
			if c.bearer != "" {
				r.Header.Set("Authorization", c.bearer)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("状态码 = %d, 期望 401", rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, c.want) {
				t.Errorf("响应正文里没有 %q: %s", c.want, body)
			}
			assertChineseError(t, c.name, body)
		})
	}
}

// TestAuthErrorConstantsLocalized 断言认证处理器使用的用户可见文案常量
// 全部含汉字且不含谚文。连无法直接用 requireAuth 打到
// (= 要经 pg 的)处理器文案，也能在没有 DB 时抓住回归。
func TestAuthErrorConstantsLocalized(t *testing.T) {
	consts := map[string]string{
		"authErrUnauthorized":         authErrUnauthorized,
		"authErrTokenInvalid":         authErrTokenInvalid,
		"authErrPasswordAlreadySet":   authErrPasswordAlreadySet,
		"authErrPasswordEmpty":        authErrPasswordEmpty,
		"authErrNewPasswordEmpty":     authErrNewPasswordEmpty,
		"authErrPasswordHash":         authErrPasswordHash,
		"authErrSaveFailedPrefix":     authErrSaveFailedPrefix,
		"authErrTokenGen":             authErrTokenGen,
		"authErrBadRequest":           authErrBadRequest,
		"authErrPasswordNotInit":      authErrPasswordNotInit,
		"authErrCurrentPasswordWrong": authErrCurrentPasswordWrong,
		"authErrBadCredential":        authErrBadCredential,
	}
	for name, msg := range consts {
		assertChineseError(t, name, msg)
	}
}
