package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// decodeErrorField 从 writeErr 的 JSON 正文里取出 "error" 字符串，
// 以便断言用户可见文案的中文属性。
func decodeErrorField(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v (正文 %s)", err, body)
	}
	return resp.Error
}

// TestTaskCategoryErrorConstantsLocalized 固定 task_categories.go 里
// 每一条用户可见错误字面量都是简体中文(含汉字、不含谚文)。
// 一旦有人把其中某条改回非中文文案，本测试即失败。
func TestTaskCategoryErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"errTaskCatRequestTooLarge": errTaskCatRequestTooLarge,
		"errTaskCatNameEmpty":       errTaskCatNameEmpty,
		"errTaskCatNameTooLong":     errTaskCatNameTooLong,
		"errTaskCatNameConflict":    errTaskCatNameConflict,
		"errTaskCatInvalidID":       errTaskCatInvalidID,
		"errTaskCatBatchSizeFmt":    fmt.Sprintf(errTaskCatBatchSizeFmt, db.MaxTaskCategoryBatchSize),
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestTaskCategoryResponsesLocalized 驱动那些从不触碰数据库的校验器，
// 确认中文文案确实进入 HTTP 正文。
// 请求解码器与 id 解析器只看正文，writeTaskCategoryError
// 映射 db 哨兵，批量大小守卫在读取 s.m 之前就返回。
func TestTaskCategoryResponsesLocalized(t *testing.T) {
	// decodeTaskCategoryRequest —— 只看正文的校验，不涉及 DB。
	decodeCase := func(name, body string) string {
		req := httptest.NewRequest(http.MethodPost, "/api/task-categories", strings.NewReader(body))
		rec := httptest.NewRecorder()
		if _, ok := decodeTaskCategoryRequest(rec, req); ok {
			t.Fatalf("%s: 校验不应通过 (status=%d)", name, rec.Code)
		}
		return decodeErrorField(t, rec.Body.Bytes())
	}
	assertChineseError(t, "name-empty", decodeCase("name-empty", `{"name":"   "}`))
	assertChineseError(t, "name-too-long",
		decodeCase("name-too-long", `{"name":"`+strings.Repeat("字", db.MaxTaskCategoryNameRunes+1)+`"}`))
	// 正文超过 maxTaskCategoryRequestBytes，因此 MaxBytesReader 在解码途中报错。
	assertChineseError(t, "too-large",
		decodeCase("too-large", `{"name":"`+strings.Repeat("a", maxTaskCategoryRequestBytes+1)+`"}`))

	// parseCategoryIDField —— 为零/负的 category_id 被拒绝。
	rec := httptest.NewRecorder()
	if _, ok := parseCategoryIDField(rec, json.RawMessage("0")); ok {
		t.Fatal("非法 id 不应通过")
	}
	assertChineseError(t, "invalid-id", decodeErrorField(t, rec.Body.Bytes()))

	// writeTaskCategoryError —— 名称冲突哨兵映射成中文 409。
	rec = httptest.NewRecorder()
	writeTaskCategoryError(rec, db.ErrTaskCategoryNameConflict)
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d, want 409", rec.Code)
	}
	assertChineseError(t, "name-conflict", decodeErrorField(t, rec.Body.Bytes()))

	// 批量移动 —— 空选择在任何 DB 访问之前就被拒绝，
	// 因此零值 Server 无需解引用 s.m 就能到达该守卫。
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/category/batch",
		strings.NewReader(`{"task_ids":[],"category_id":null}`))
	rec = httptest.NewRecorder()
	(&Server{}).updateTasksCategoryBatch(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty batch status=%d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
	assertChineseError(t, "batch-size", decodeErrorField(t, rec.Body.Bytes()))
}
