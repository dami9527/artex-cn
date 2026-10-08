//go:build !embedui

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWebUIStubErrorLocalized 守卫未内嵌 stub 的 HTTP 响应：
// 默认(!embedui)构建服务的请求必须得到中文 404 正文，而不是改版前的
// 旧文案。这是 server/ 中此前只扫 `writeErr(` 的清查漏掉的最后一个
// 用户可见 HTTP 错误响应(它用的是 http.Error，不是 writeErr)。若上游
// 重新同步时退回了旧字面量，本测试即失败。
func TestWebUIStubErrorLocalized(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	(&Server{}).webuiHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("状态码不是 404: %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	assertChineseError(t, "webui.stub", body)
	// 命令引用必须原样保留(不在翻译范围)。
	for _, want := range []string{"next dev", "-tags embedui"} {
		if !strings.Contains(body, want) {
			t.Fatalf("响应里没有命令引用 %q: %q", want, body)
		}
	}
}
