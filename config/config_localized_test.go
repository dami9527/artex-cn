package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hasHan 判断 s 是否含有 CJK 汉字（本仓库中文文案的特征）。用于确认安装路径上
// 面向用户的文案确实是中文。
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// hasHangul 判断 s 是否含有谚文音节。本仓库是中文版，任何面向用户的文案都不得出现。
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xac00 && r <= 0xd7a3 {
			return true
		}
	}
	return false
}

// assertChinese 断言文案是简体中文且不含谚文：必须出现汉字，且不得出现任何谚文。
// 标识符（ARTEX_PG_DSN、dsn、host/user/dbname 之类）是 ASCII，天然不受影响。
func assertChinese(t *testing.T, label, s string) {
	t.Helper()
	if hasHangul(s) {
		t.Errorf("%s: Hangul remains: %q", label, s)
	}
	if !hasHan(s) {
		t.Errorf("%s: no Han ideograph found (expected Chinese): %q", label, s)
	}
}

// TestPostgresDSNErrorLocalized 把安装路径上的启动错误钉在中文上。
// 当 ARTEX_PG_DSN 与配置文件都没提供数据库时，PostgresDSN 返回的错误会被原样向上传播
// （db.DSN → server.NewManager → cmd/artex/main.go 的 `log.Fatalf("open stores: %v", err)`）。
// 缺少配置就启动的人最先看到它，因此必须是中文。
func TestPostgresDSNErrorLocalized(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARTEX_PG_DSN", "")
	t.Setenv("ARTEX_CONFIG", filepath.Join(dir, "nope.json"))

	_, _, err := PostgresDSN()
	if err == nil {
		t.Fatal("missing config should error")
	}
	assertChinese(t, "startup error", err.Error())

	// 运维需要据此排查的标识符保持原样（不翻译）。
	for _, want := range []string{"ARTEX_PG_DSN", "database", "dsn", "host/user/dbname"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("startup error should keep identifier %q verbatim: %q", want, err.Error())
		}
	}

	// 记录运维最终看到的形态（整条路径上没有 %w 包装），便于在上下文里人工核对。
	t.Logf("open stores: %v", err)
}

// TestPostgresDSNSourceLocalized 把三个来源标签钉在中文上（它们记在 server/manager.go
// 的日志里）。这些标签与启动错误在同一个函数里，一起本地化可以避免同一个文件里混用语言。
func TestPostgresDSNSourceLocalized(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// 环境变量来源
	t.Setenv("ARTEX_CONFIG", filepath.Join(dir, "nope.json"))
	t.Setenv("ARTEX_PG_DSN", "postgres://envwins/x")
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("env source: %v", err)
	} else {
		assertChinese(t, "env source", source)
	}

	// 配置文件 dsn 来源
	t.Setenv("ARTEX_PG_DSN", "")
	if err := os.WriteFile(cfgPath, []byte(`{"database":{"dsn":"postgres://full/dsn"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARTEX_CONFIG", cfgPath)
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("dsn source: %v", err)
	} else {
		assertChinese(t, "dsn source", source)
	}

	// 配置文件字段来源
	if err := os.WriteFile(cfgPath, []byte(`{"database":{"host":"10.1.2.3","dbname":"d","user":"u"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("fields source: %v", err)
	} else {
		assertChinese(t, "fields source", source)
	}
}
