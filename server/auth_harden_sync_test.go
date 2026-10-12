package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// 上游 a951e4a 移植（待办：认证读取 fail-closed + 首次设置原子化 + 密码长度上限）的
// 回归防御。GetSetting 的「键不存在」与「读取失败」只差一个 error 返回值（两者的
// value 都是 ""）。如果密码处理器把 error 当成「未设置」，数据库故障期间初始化入口就
// 会被打开。下面两个测试关闭连接池来制造读取失败，确认 fail-closed（503）。
func TestAuthStatusFailsClosedWhenDataSourceUnavailable(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	if err := m.pg.Close(); err != nil {
		t.Fatal(err)
	}

	s := &Server{m: m}
	w := httptest.NewRecorder()
	s.authStatus(w, httptest.NewRequest("GET", "/api/auth/status", nil))

	if w.Code != 503 {
		t.Fatalf("status=%d，期望 503（把读取失败当成未初始化，用户就会去 /setup 覆盖密码）；body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		Initialized *bool `json:"initialized"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err == nil && payload.Initialized != nil {
		t.Fatalf("读取失败时不得返回 initialized: %v", *payload.Initialized)
	}
}

func TestAuthInitFailsClosedWhenDataSourceUnavailable(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	if err := m.pg.Close(); err != nil {
		t.Fatal(err)
	}

	s := &Server{m: m}
	w := httptest.NewRecorder()
	body := strings.NewReader(`{"password":"correct horse battery","setup_token":"x"}`)
	s.authInit(w, httptest.NewRequest("POST", "/api/auth/init", body))

	if w.Code != 503 {
		t.Fatalf("status=%d，期望 503（读取失败时放行，未认证请求会覆盖现有密码）；body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "token") {
		t.Fatalf("读取失败时不得签发 token: %s", w.Body.String())
	}
}

// 确认服务器强制密码字节上限（bcrypt 72 字节）。73 字节必须被以 400 拒绝
// （此前 bcrypt 会在哈希阶段失败并返回 500）。
func TestAuthInitRejectsTooLongPasswordDB(t *testing.T) {
	rig := newAuthTestRig(t)
	tok := rig.s.currentSetupToken()

	code, body := rig.do("POST", "/api/auth/init", "", map[string]any{
		"password":    strings.Repeat("a", 73),
		"setup_token": tok,
	})
	rig.expect(400, code, body, authErrPasswordTooLong)
}
