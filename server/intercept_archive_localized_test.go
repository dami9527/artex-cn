package server

import (
	"net/url"
	"testing"
	"unicode"
)

// assertChineseError 在 msg 为空、不含汉字(= 未本地化的非中文文案)或仍残留谚文时失败。
// 汉字与 ASCII 字段名/枚举值都是允许的；缺少汉字或出现谚文才算不合格。
func assertChineseError(t *testing.T, label, msg string) {
	t.Helper()
	if msg == "" {
		t.Fatalf("%s: 空文案", label)
	}
	hasHan := false
	for _, r := range msg {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s: 仍有谚文残留: %q", label, msg)
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	if !hasHan {
		t.Fatalf("%s: 未检测到汉字: %q", label, msg)
	}
}

// TestInterceptArchiveErrorsLocalized 是第一束 F3 的守卫：intercept.go 与
// task_archives.go 里的纯校验函数产出的每一条用户可见错误响应都必须是简体中文
// (含汉字、不含谚文)。一旦有人把其中某个字面量改回非中文文案，本测试即失败。
func TestInterceptArchiveErrorsLocalized(t *testing.T) {
	// intercept.go: interceptFilterParams — 查询串校验。
	if _, err := interceptFilterParams(url.Values{"status": {"bogus"}}); err == nil {
		t.Fatal("status 校验不应通过")
	} else {
		assertChineseError(t, "filter.status", err.Error())
	}
	if _, err := interceptFilterParams(url.Values{"decision_source": {"bogus"}}); err == nil {
		t.Fatal("decision_source 校验不应通过")
	} else {
		assertChineseError(t, "filter.decision_source", err.Error())
	}

	// intercept.go: validateInterceptRuleReq —— 通过让前面的字段合法
	// 来隔离每个分支。
	ruleCases := []struct {
		label string
		req   interceptRuleReq
	}{
		{"rule.name", interceptRuleReq{Name: ""}},
		{"rule.match_target", interceptRuleReq{Name: "r", MatchTarget: "bad"}},
		{"rule.match_type", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "bad"}},
		{"rule.pattern_empty", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "string", Pattern: ""}},
		{"rule.action", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "string", Pattern: "p", Action: "bad"}},
		{"rule.regex", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "regex", Pattern: "(", Action: "allow"}},
	}
	for _, tc := range ruleCases {
		err := validateInterceptRuleReq(tc.req)
		if err == nil {
			t.Fatalf("%s: 校验不应通过", tc.label)
		}
		assertChineseError(t, tc.label, err.Error())
	}

	// task_archives.go: normalizeArchiveIDs —— 批量 id 校验。
	if _, err := normalizeArchiveIDs(nil); err == nil {
		t.Fatal("空 archive_ids 校验不应通过")
	} else {
		assertChineseError(t, "archive_ids.empty", err.Error())
	}
	tooMany := make([]int64, 101)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	if _, err := normalizeArchiveIDs(tooMany); err == nil {
		t.Fatal("101 个 archive_ids 校验不应通过")
	} else {
		assertChineseError(t, "archive_ids.too_many", err.Error())
	}
	if _, err := normalizeArchiveIDs([]int64{0}); err == nil {
		t.Fatal("0 号 archive id 校验不应通过")
	} else {
		assertChineseError(t, "archive_ids.nonpositive", err.Error())
	}

	// task_archives.go: validateArchivePath —— 归档包路径校验。
	if err := validateArchivePath("/tmp/artex-data", ""); err == nil {
		t.Fatal("空归档路径校验不应通过")
	} else {
		assertChineseError(t, "archive_path.empty", err.Error())
	}
	if err := validateArchivePath("/tmp/artex-data", "/etc/passwd"); err == nil {
		t.Fatal("管理目录之外的路径校验不应通过")
	} else {
		assertChineseError(t, "archive_path.outside", err.Error())
	}
}
