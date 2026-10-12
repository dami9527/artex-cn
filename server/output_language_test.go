package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Autumn-27/artex/config"
)

// TestOutputLanguageCacheFollowsSettings 钉住缓存的输出语言（每次 writeErr 与标签都要
// 读它）会被 PUT /api/settings 刷新：切换语言在下一次响应就生效，而不必每条消息都去读
// settings 表。
func TestOutputLanguageCacheFollowsSettings(t *testing.T) {
	t.Setenv("ARTEX_LANG", "")
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	prev, hadPrev, _ := m.pg.GetSetting(settingOutputLanguage)
	t.Cleanup(func() { config.CurrentLanguage = nil })
	// 用 defer（而不是 t.Cleanup）保证恢复动作先于后面 defer 的 m.Close 执行。
	defer func() {
		if hadPrev {
			_ = m.pg.SetSetting(settingOutputLanguage, prev)
		} else {
			_, _ = m.pg.Exec(`DELETE FROM settings WHERE key=$1`, settingOutputLanguage)
		}
	}()
	_, _ = m.pg.Exec(`DELETE FROM settings WHERE key=$1`, settingOutputLanguage)

	s := New(context.Background(), m, t.TempDir(), t.TempDir(), t.TempDir())
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	do := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec
	}
	errOf := func(rec *httptest.ResponseRecorder) string {
		var v struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &v)
		return v.Error
	}

	if got := s.outputLanguage(); got != "zh" {
		t.Fatalf("默认输出语言 = %q，期望 zh", got)
	}
	if got := errOf(do(http.MethodPut, "/api/settings", map[string]any{"output_language": "fr"})); got != errOutputLanguageInvalid {
		t.Fatalf("中文错误文案 = %q，期望 %q", got, errOutputLanguageInvalid)
	}

	if rec := do(http.MethodPut, "/api/settings", map[string]any{"output_language": "en"}); rec.Code != http.StatusOK {
		t.Fatalf("切换到 en 时 status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := errOf(do(http.MethodPut, "/api/settings", map[string]any{"output_language": "fr"})); got == errOutputLanguageInvalid || got == "" {
		t.Fatalf("切换到 en 之后错误文案应被翻译，实际得到 %q", got)
	}
	var settings struct {
		OutputLanguage string `json:"output_language"`
	}
	_ = json.Unmarshal(do(http.MethodGet, "/api/settings", nil).Body.Bytes(), &settings)
	if settings.OutputLanguage != "en" {
		t.Fatalf("GET /api/settings 的 output_language = %q，期望 en", settings.OutputLanguage)
	}
}
