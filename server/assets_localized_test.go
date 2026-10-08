package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAssetErrorConstantsLocalized 钉住 task_assets.go 与 assets.go 里所有面向用户的
// writeErr 文案：必须是简体中文（含汉字、不含谚文）。谁把它改回别的语言，这里就会失败。
func TestAssetErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"errTaskAssetRequestTooLarge": errTaskAssetRequestTooLarge,
		"errTaskAssetScopeConflict":   errTaskAssetScopeConflict,
		"errCompanyRequestTooLarge":   errCompanyRequestTooLarge,
		"errCompanyNameConflict":      errCompanyNameConflict,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestTaskAssetProvenanceLocalized 钉住资产来源摘要标签：它们会被写进
// task_asset_links.source_summary，并在任务详情的 session/资产标签页里原样渲染。
// 任何一个改回别的语言都会在这里失败。手工新增那一版在 db 包里
// （db.manualTaskScopeSummary），由那边的测试负责钉住。
func TestTaskAssetProvenanceLocalized(t *testing.T) {
	cases := map[string]string{
		"taskAssetSourceAPISummary":  taskAssetSourceAPISummary,
		"taskAssetSourceTaskSummary": taskAssetSourceTaskSummary,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestAssetResponsesLocalized 驱动那些在访问数据库之前就会返回的请求校验路径，
// 确认中文文案确实落进了 HTTP 响应体。这些路径都不需要真实数据：scope/asset_ids
// 冲突判断在读取 s.m.Assets() 之前返回，MaxBytesReader 在解码中途触发，
// decodeCompanyMutationRequest 只依赖请求体，所以一个只持有 task map 的 Manager
// 就能走到每一处。
func TestAssetResponsesLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"1": {ID: "1"}}}}

	// attachTaskAssets —— scope 与 asset_ids 不允许同时提交。
	conflictReq := httptest.NewRequest(http.MethodPost, "/api/tasks/1/assets",
		strings.NewReader(`{"scope":["example.com"],"asset_ids":[1]}`))
	conflictReq.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	s.attachTaskAssets(rec, conflictReq)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("scope+asset_ids status=%d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
	assertChineseError(t, "scope-conflict", decodeErrorField(t, rec.Body.Bytes()))

	// attachTaskAssets —— 超大请求体在解码时触发 MaxBytesReader。
	bigReq := httptest.NewRequest(http.MethodPost, "/api/tasks/1/assets",
		strings.NewReader(`{"source_summary":"`+strings.Repeat("a", maxTaskAssetRequestBytes+1)+`"}`))
	bigReq.SetPathValue("id", "1")
	rec = httptest.NewRecorder()
	s.attachTaskAssets(rec, bigReq)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("task asset too-large status=%d, want 413 (body %s)", rec.Code, rec.Body.String())
	}
	assertChineseError(t, "task-asset-too-large", decodeErrorField(t, rec.Body.Bytes()))

	// decodeCompanyMutationRequest —— 只依赖请求体的辅助函数，不需要 DB 或 Server 状态。
	compReq := httptest.NewRequest(http.MethodPost, "/api/companies",
		strings.NewReader(`{"name":"`+strings.Repeat("a", maxCompanyMutationBodyBytes+1)+`"}`))
	rec = httptest.NewRecorder()
	var dst map[string]any
	if ok := decodeCompanyMutationRequest(rec, compReq, &dst); ok {
		t.Fatalf("company too-large body should not decode (status=%d)", rec.Code)
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("company too-large status=%d, want 413 (body %s)", rec.Code, rec.Body.String())
	}
	assertChineseError(t, "company-too-large", decodeErrorField(t, rec.Body.Bytes()))
}
