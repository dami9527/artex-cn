package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防御测试：保证 customtool.go 的自定义工具 CRUD·试运行端点经 writeErr
// 返回的校验错误响应为简体中文。中文判定复用 F3a 的
// assertChineseError(含汉字·无谚文)，响应正文提取复用 task_categories 测试的
// decodeErrorField(同属 package server)。智能体读取的 actool.Errorf 工具
// 结果与工具 schema description 属于大脑边界，不在翻译范围，本测试也不涉及。

// TestCustomToolErrorConstantsLocalized 断言经 writeErr 暴露的 8 个常量
// 全部为简体中文。其中两个(errCustomToolKeyExists·errCustomToolEditCustomOnly)需要处理器
// 先经 pg.GetTool(DB) 才能到达，下面的真实 HTTP 测试触及不到。这里用常量断言
// 固定，至于「哪个分支用到这个常量」已通过代码路径追踪确认
// (与 goals_api·task_control 的 DB 路径检查密度一致)。
func TestCustomToolErrorConstantsLocalized(t *testing.T) {
	for _, c := range []struct{ name, msg string }{
		{"keyFormat", errCustomToolKeyFormat},
		{"kindInvalid", errCustomToolKindInvalid},
		{"httpSchemaRequired", errCustomToolHTTPSchemaRequired},
		{"keyExists", errCustomToolKeyExists},
		{"editCustomOnly", errCustomToolEditCustomOnly},
		{"badBody", errCustomToolBadBody},
		{"shellNoExec", errCustomToolShellNoExec},
		{"unknownKindPrefix", errCustomToolUnknownKindPrefix},
	} {
		assertChineseError(t, "customtool."+c.name, c.msg)
	}
}

// newCustomToolServer 构造一个能通过 pg 关卡(s.pg)、但没有真实 DB 连接的 Server。
// &db.DB{} 内嵌的 *sql.DB 为 nil，但指针本身 non-nil，因此 pg() 不会写出 503，
// 而是原样返回。下面这些校验分支全部在 pg.GetTool 之类的真实 DB 调用「之前」
// 返回，不会解引用 nil DB(已按代码路径确认)。
func newCustomToolServer() *Server {
	return &Server{m: &Manager{pg: &db.DB{}}}
}

// TestCustomToolCreateValidationLocalized 用真实 HTTP 跑 pgCreateCustomTool 的
// 三个输入校验路径，确认响应正文为简体中文。三条路径都在 pg.GetTool(第 68 行)之前
// 返回。
func TestCustomToolCreateValidationLocalized(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"bad-key", `{"key":"BadKey","kind":"command"}`, errCustomToolKeyFormat},
		{"bad-kind", `{"key":"goodkey","kind":"bogus"}`, errCustomToolKindInvalid},
		{"http-no-schema", `{"key":"goodkey","kind":"http"}`, errCustomToolHTTPSchemaRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCustomToolServer()
			req := httptest.NewRequest(http.MethodPost, "/api/tools/custom", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			s.pgCreateCustomTool(rec, req)
			if rec.Code != 400 && rec.Code != 409 {
				t.Fatalf("期望校验错误状态码，实际 %d (正文 %q)", rec.Code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("响应正文不一致: got %q want %q", got, c.want)
			}
			assertChineseError(t, "customtool.create."+c.name, got)
		})
	}
}

// TestCustomToolTestRunValidationLocalized 用真实 HTTP 跑 pgTestCustomTool 的
// 三个输入校验路径，确认响应正文为简体中文。三者都没有 DB 调用。
func TestCustomToolTestRunValidationLocalized(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"bad-body", `not-json`, errCustomToolBadBody},
		{"shell", `{"kind":"shell"}`, errCustomToolShellNoExec},
		{"unknown-kind", `{"kind":"bogus"}`, errCustomToolUnknownKindPrefix + "bogus"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCustomToolServer()
			req := httptest.NewRequest(http.MethodPost, "/api/tools/custom/test", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			s.pgTestCustomTool(rec, req)
			if rec.Code != 400 {
				t.Fatalf("期望状态码 400，实际 %d (正文 %q)", rec.Code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("响应正文不一致: got %q want %q", got, c.want)
			}
			// unknown-kind 后面会拼上 req.Kind("bogus") 的拉丁尾巴，因此只检查不含谚文。
			assertChineseError(t, "customtool.test."+c.name, errCustomToolUnknownKindPrefix)
		})
	}
}
