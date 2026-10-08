package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestSyncScopeSentryConstantsLocalized 是 F3b(sync_scopesentry.go) 的守卫：
// 每一条用户可见消息常量 —— 无论经 writeErr 的 HTTP 正文暴露，
// 还是出现在同步结果的 warnings/errors JSON 数组里 —— 都必须是简体中文(含汉字、
// 不含谚文)。任何字面量改回非中文文案都会让本测试失败。
func TestSyncScopeSentryConstantsLocalized(t *testing.T) {
	cases := []struct {
		label string
		msg   string
	}{
		{"errSSDataSourceMissingFmt", errSSDataSourceMissingFmt},
		{"errSSDataSourceNoURLFmt", errSSDataSourceNoURLFmt},
		{"errSSListProjectsPrefix", errSSListProjectsPrefix},
		{"errSSParseProjectsPrefix", errSSParseProjectsPrefix},
		{"errSSListTasksPrefix", errSSListTasksPrefix},
		{"errSSParseTasksPrefix", errSSParseTasksPrefix},
		{"errSSDimension", errSSDimension},
		{"errSSTargetsEmpty", errSSTargetsEmpty},
		{"warnSSProjectMetaFmt", warnSSProjectMetaFmt},
		{"warnSSCompanyCreateFmt", warnSSCompanyCreateFmt},
		{"warnSSUnknownAssetPrefix", warnSSUnknownAssetPrefix},
		{"errSSFetchFmt", errSSFetchFmt},
		{"warnSSTruncatedFmt", warnSSTruncatedFmt},
		{"errSSParseSubdomainPrefix", errSSParseSubdomainPrefix},
		{"errSSParseAppPrefix", errSSParseAppPrefix},
		{"errSSParseServicePrefix", errSSParseServicePrefix},
	}
	for _, c := range cases {
		assertChineseError(t, c.label, c.msg)
	}

	// dimension/targets 校验保留各自的 JSON 字段名与枚举值。
	if !strings.Contains(errSSDimension, "project") || !strings.Contains(errSSDimension, "task") {
		t.Fatalf("errSSDimension 丢了枚举值: %q", errSSDimension)
	}
}

// TestSyncScopeSentryFormattedLocalized 走同步处理器实际使用的
// Sprintf/Errorf 格式串，这样动词不匹配或字面量被改回旧文案，
// 都会因为跑了真实的格式化操作而被抓住(而不只是检查常量)。
func TestSyncScopeSentryFormattedLocalized(t *testing.T) {
	ds := fmt.Errorf(errSSDataSourceMissingFmt, scopeSentryMCPName).Error()
	assertChineseError(t, "scopeSentryClient/missing", ds)
	if !strings.Contains(ds, "ScopeSentry") {
		t.Fatalf("数据源名称不见了: %q", ds)
	}

	noURL := fmt.Errorf(errSSDataSourceNoURLFmt, scopeSentryMCPName).Error()
	assertChineseError(t, "scopeSentryClient/noURL", noURL)
	if !strings.Contains(noURL, "URL") {
		t.Fatalf("URL 记号不见了: %q", noURL)
	}

	meta := fmt.Sprintf(warnSSProjectMetaFmt, "proj-1", errors.New("boom"))
	assertChineseError(t, "warnSSProjectMetaFmt", meta)
	if !strings.Contains(meta, "proj-1") || !strings.Contains(meta, "boom") {
		t.Fatalf("项目/原始错误没有被代入: %q", meta)
	}

	company := fmt.Sprintf(warnSSCompanyCreateFmt, "ACME", errors.New("boom"))
	assertChineseError(t, "warnSSCompanyCreateFmt", company)
	if !strings.Contains(company, "ACME") {
		t.Fatalf("企业名称没有被代入: %q", company)
	}

	fetch := fmt.Sprintf(errSSFetchFmt, "subdomain", "t1", errors.New("boom"))
	assertChineseError(t, "errSSFetchFmt", fetch)
	if !strings.Contains(fetch, "subdomain") || !strings.Contains(fetch, "t1") {
		t.Fatalf("资产类型/对象没有被代入: %q", fetch)
	}

	truncated := fmt.Sprintf(warnSSTruncatedFmt, "app", "t1", syncMaxPerType)
	assertChineseError(t, "warnSSTruncatedFmt", truncated)
	if !strings.Contains(truncated, "5000") {
		t.Fatalf("上限条数没有被代入: %q", truncated)
	}
}

// TestSyncScopeSentryIngestParseErrorsLocalized 对每种资产类型用损坏的
// JSON 跑 ssIngest。反序列化失败在任何 store 访问之前返回，因此这是
// 真实代码路径(nil AssetStore 从不解引用)，
// 把解析错误文案固定为中文。
func TestSyncScopeSentryIngestParseErrorsLocalized(t *testing.T) {
	s := &Server{}
	bad := json.RawMessage("not json")
	for _, kind := range []string{"subdomain", "app", "service"} {
		got := s.ssIngest(nil, kind, bad, map[string]int{})
		if got == "" {
			t.Fatalf("%s: JSON 已损坏，错误字符串却为空", kind)
		}
		assertChineseError(t, "ssIngest/"+kind, got)
		if !strings.Contains(got, kind) {
			t.Fatalf("%s: 资产类型前缀不见了: %q", kind, got)
		}
	}
}
