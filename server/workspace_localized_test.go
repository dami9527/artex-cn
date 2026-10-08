package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestWorkspaceErrorConstantsLocalized 是 F3b(workspace.go) 的守卫：工作区文件管理器
// 返回的每一条用户可见错误字符串都必须是简体中文(含汉字、
// 不含谚文)。任何字面量改回非中文文案都会让本测试失败。这些常量
// 不依赖 DB，因此总会运行(不需要 postgres)。
func TestWorkspaceErrorConstantsLocalized(t *testing.T) {
	for label, msg := range map[string]string{
		"illegalPath":      errWsIllegalPath,
		"pathNotFound":     errWsPathNotFound,
		"notDir":           errWsNotDir,
		"fileNotFound":     errWsFileNotFound,
		"isDir":            errWsIsDir,
		"targetIsDir":      errWsTargetIsDir,
		"cannotDeleteRoot": errWsCannotDeleteRoot,
		"uploadDirMissing": errWsUploadDirMissing,
		"uploadParse":      errWsUploadParse,
		"noUploadFile":     errWsNoUploadFile,
	} {
		assertChineseError(t, label, msg)
	}
}

// wsErrBody 取出 writeErr 产生的 {"error": "..."} 字符串。
func wsErrBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v (正文 %q)", err, rec.Body.String())
	}
	return out.Error
}

// TestWorkspaceHandlerResponsesLocalized 用真实 HTTP 驱动工作区
// 文件管理器处理器。它们只访问 s.m.dir 与文件系统(从不碰 s.m.pg)，
// 因此一个临时目录 Manager 就够 —— 不需要 DB。这证明中文常量确实
// 进入了 HTTP 响应正文，而不只是常量本身是中文。
func TestWorkspaceHandlerResponsesLocalized(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &Server{m: &Manager{dir: dir}}

	cases := []struct {
		name    string
		handler http.HandlerFunc
		req     *http.Request
		code    int
		want    string
	}{
		{"read-missing", s.wsRead, httptest.NewRequest(http.MethodGet, "/api/workspace/read?path=nope.txt", nil), 404, errWsFileNotFound},
		{"read-dir", s.wsRead, httptest.NewRequest(http.MethodGet, "/api/workspace/read?path=sub", nil), 400, errWsIsDir},
		{"list-missing", s.wsList, httptest.NewRequest(http.MethodGet, "/api/workspace/list?path=nope", nil), 404, errWsPathNotFound},
		{"list-not-dir", s.wsList, nil, 400, errWsNotDir}, // req built below (needs a real file)
		{"delete-root", s.wsDelete, httptest.NewRequest(http.MethodDelete, "/api/workspace/delete?path=", nil), 400, errWsCannotDeleteRoot},
	}

	// list-not-dir 需要一个真实存在的文件来指向。
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases[3].req = httptest.NewRequest(http.MethodGet, "/api/workspace/list?path=a.txt", nil)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.handler(rec, c.req)
			if rec.Code != c.code {
				t.Fatalf("status = %d, want %d (正文 %q)", rec.Code, c.code, rec.Body.String())
			}
			got := wsErrBody(t, rec)
			if got != c.want {
				t.Fatalf("error = %q, want %q", got, c.want)
			}
			assertChineseError(t, c.name, got)
		})
	}
}
