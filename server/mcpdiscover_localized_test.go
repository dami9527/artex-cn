package server

import (
	"context"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防御测试：保证 mcpdiscover.go 的 connectMCP 返回的传输配置校验错误文案
// 为简体中文。四条错误路径(stdio 缺命令·http 缺 URL·sse 缺 URL·未知
// 传输方式)全都在真正连接(dial)服务器之前返回，因此不需要 DB·网络，
// 直接调用 connectMCP 即可检查返回文案。中文判定复用 F3a 的
// assertChineseError(含汉字·无谚文)。传输方式 enum(stdio/http/sse) 与
// URL 是线上标识符，保持 ASCII，不会被汉字判定拦住。一旦有人把这条文案
// 改回非中文，常量固定与真实路径会一起失败。
func TestConnectMCPErrorsLocalized(t *testing.T) {
	// 常量固定
	for _, c := range []struct{ label, msg string }{
		{"stdio_no_command", errMCPStdioNoCommand},
		{"http_no_url", errMCPHTTPNoURL},
		{"sse_no_url", errMCPSSENoURL},
		{"unknown_transport", errMCPUnknownTransportFmt},
	} {
		assertChineseError(t, c.label, c.msg)
	}

	// 真实路径: 四个拒绝分支都在 dial 之前返回。
	for _, c := range []struct {
		label string
		srv   *db.MCPServer
	}{
		{"stdio_missing_command", &db.MCPServer{Transport: "stdio", Command: ""}},
		{"http_missing_url", &db.MCPServer{Transport: "http", URL: ""}},
		{"sse_missing_url", &db.MCPServer{Transport: "sse", URL: ""}},
		{"unknown_transport_path", &db.MCPServer{Transport: "websocket"}},
	} {
		_, err := connectMCP(context.Background(), c.srv)
		if err == nil {
			t.Fatalf("%s: 没有报错(校验通过了)", c.label)
		}
		assertChineseError(t, c.label, err.Error())
	}
}
