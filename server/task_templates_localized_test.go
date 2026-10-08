package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// TestTaskTemplateErrorConstantsLocalized 是 F3b 任务模板这一束的守卫：
// 任务模板 API 返回的每一条用户可见字符串都必须是简体中文，
// 不得残留谚文。格式串常量在格式化之后检查，
// 以便 %s/%d 动词解析成具体文案。
func TestTaskTemplateErrorConstantsLocalized(t *testing.T) {
	assertChineseError(t, "request-too-large", errTaskTemplateRequestTooLarge)
	assertChineseError(t, "name-conflict", errTaskTemplateNameConflict)
	assertChineseError(t, "rule-invalid", errTaskTemplateRuleInvalid)
	assertChineseError(t, "no-fields", errTaskTemplateNoFields)
	assertChineseError(t, "field-too-long",
		fmt.Sprintf(errTaskTemplateFieldTooLongFmt, "name", db.MaxTaskTemplateNameRunes))
}

// TestTaskTemplateResponsesLocalized 驱动那些不触碰数据库的真实响应路径：
// 正文解码器拒绝超限请求，字段
// 校验器拒绝超长名称，writeTaskTemplateErr 把
// 名称冲突哨兵映射成中文 409 正文。
func TestTaskTemplateResponsesLocalized(t *testing.T) {
	// decodeTaskTemplateRequest —— 正文超限时 MaxBytesReader 在任何 DB 访问
	// 之前就报错，因此 413 文案是独立产生的。
	body := `{"name":"` + strings.Repeat("a", maxTaskTemplateRequestBytes+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/task-templates", strings.NewReader(body))
	rec := httptest.NewRecorder()
	var decoded taskTemplateRequest
	if _, ok := decodeTaskTemplateRequest(rec, req, &decoded); ok {
		t.Fatalf("too-large: 解码不应通过 (status=%d)", rec.Code)
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("too-large status=%d, want 413", rec.Code)
	}
	assertChineseError(t, "too-large", decodeErrorField(t, rec.Body.Bytes()))

	// validateTaskTemplateRequest —— 名称超过字符上限时返回
	// 中文字段超长错误(处理器经 writeErr 暴露)。
	longName := strings.Repeat("字", db.MaxTaskTemplateNameRunes+1)
	err := validateTaskTemplateRequest(taskTemplateRequest{Name: &longName})
	if err == nil {
		t.Fatal("name-too-long: 校验不应通过")
	}
	assertChineseError(t, "name-too-long", err.Error())

	// writeTaskTemplateErr —— 名称冲突哨兵映射成中文 409。
	rec = httptest.NewRecorder()
	writeTaskTemplateErr(rec, db.ErrTaskTemplateNameConflict)
	if rec.Code != http.StatusConflict {
		t.Fatalf("name-conflict status=%d, want 409", rec.Code)
	}
	assertChineseError(t, "name-conflict-response", decodeErrorField(t, rec.Body.Bytes()))
}
