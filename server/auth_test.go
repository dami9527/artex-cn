package server

import (
	"strings"
	"testing"
)

// TestValidatePassword 钉住密码边界：下限按**字符数**（与 /setup 页的前端规则一致），
// 上限按**字节数**（bcrypt 的硬限制，超过 72 字节 GenerateFromPassword 直接报
// ErrPasswordTooLong）。写成表驱动是为了把「汉字按字符数而非字节数计」这条容易写错的
// 语义固定下来。
//
// 注：数据库故障时两个 handler 必须 fail closed（而不是把「读不到」当成「没设置」）
// 的回归防护由 auth_harden_sync_test.go 承担，那里用真实的连接池故障来复现。
func TestValidatePassword(t *testing.T) {
	for _, tc := range []struct {
		name, pw string
		wantErr  bool
	}{
		{"空", "", true},
		{"七位", "1234567", true},
		{"八位", "12345678", false},
		{"八个汉字按字符数而非字节数计", "密码密码密码密码", false},
		{"三个汉字够 9 字节但只有 3 个字符", "密码强", true},
		{"72 字节", strings.Repeat("a", 72), false},
		{"73 字节超出 bcrypt 上限", strings.Repeat("a", 73), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validatePassword(tc.pw); (got != "") != tc.wantErr {
				t.Fatalf("validatePassword(%q)=%q, wantErr=%v", tc.pw, got, tc.wantErr)
			}
		})
	}
}
