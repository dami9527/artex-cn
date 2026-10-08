package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServerMgmtConstantsLocalized 固定从 server_mgmt.go 抽出的
// 每一条用户可见错误/响应字面量都是简体中文(含汉字、
// 不含谚文)。任何一条改回非中文文案都会让 assertChineseError
// 因谚文残留而失败。assertChineseError / decodeErrorField 复用 F3 测试套件
// (同属 package server)。
func TestServerMgmtConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"errMgmtPGUnavailable":      errMgmtPGUnavailable,
		"errMgmtTaskIDInvalid":      errMgmtTaskIDInvalid,
		"errMgmtTaskDeleting":       errMgmtTaskDeleting,
		"errMgmtTaskHasAgents":      errMgmtTaskHasAgents,
		"errMgmtAgentKeyFormat":     errMgmtAgentKeyFormat,
		"errMgmtNameEmpty":          errMgmtNameEmpty,
		"errMgmtAgentKeyExists":     errMgmtAgentKeyExists,
		"errMgmtBuiltinNoEditMeta":  errMgmtBuiltinNoEditMeta,
		"errMgmtBuiltinNoDelete":    errMgmtBuiltinNoDelete,
		"errMgmtLLMProfileIDFormat": errMgmtLLMProfileIDFormat,
		"errMgmtLLMProfileInvalid":  errMgmtLLMProfileInvalid,
		"errMgmtNoBuiltinPrompt":    errMgmtNoBuiltinPrompt,
		"errMgmtToolNotFound":       errMgmtToolNotFound,
		"errMgmtNotBuiltinTool":     errMgmtNotBuiltinTool,
		"errMgmtMCPNotFound":        errMgmtMCPNotFound,
		"errMgmtToolDiscoverFail":   errMgmtToolDiscoverFail,
		"errMgmtSkillNameInvalid":   errMgmtSkillNameInvalid,
		"errMgmtSkillNoFile":        errMgmtSkillNoFile,
		"errMgmtSkillNoMDInZip":     errMgmtSkillNoMDInZip,
		"errMgmtSkillReadMDFail":    errMgmtSkillReadMDFail,
		"errMgmtSkillNamePre":       errMgmtSkillNamePre,
		"errMgmtSkillNamePost":      errMgmtSkillNamePost,
		"errMgmtSkillExistsPre":     errMgmtSkillExistsPre,
		"errMgmtSkillExistsPost":    errMgmtSkillExistsPost,
		"errMgmtSkillZipBadPath":    errMgmtSkillZipBadPath,
		"errMgmtSkillZipTooMany":    errMgmtSkillZipTooMany,
		"errMgmtSkillFileTooLarge":  errMgmtSkillFileTooLarge,
		"errMgmtSkillZipTooLarge":   errMgmtSkillZipTooLarge,
		"errMgmtSkillNoMDAfter":     errMgmtSkillNoMDAfter,
		"errMgmtSkillInstallFail":   errMgmtSkillInstallFail,
		"errMgmtLLMActiveDelete":    errMgmtLLMActiveDelete,
		"errMgmtLLMRefChanged":      errMgmtLLMRefChanged,
		"errMgmtLLMRefTimeout":      errMgmtLLMRefTimeout,
		"errMgmtNoAPIKey":           errMgmtNoAPIKey,
		"errMgmtBuildReqFail":       errMgmtBuildReqFail,
		"errMgmtReqFail":            errMgmtReqFail,
		"errMgmtParseRespFail":      errMgmtParseRespFail,
		"errMgmtNoModelList":        errMgmtNoModelList,
		"errMgmtTmplSyntax":         errMgmtTmplSyntax,
		"errMgmtTmplVarPre":         errMgmtTmplVarPre,
		"errMgmtTmplVarPost":        errMgmtTmplVarPost,
		// 格式串用真实格式化结果检查(中文 + verb 保留)。
		"errMgmtAPIReturned": fmt.Sprintf(errMgmtAPIReturned, 404, "detail"),
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestServerMgmtAPIReturnedFormat 守卫模型列表探针的格式串：
// 必须保留状态码与正文占位符(把 %d 改成错误的动词就会丢掉
// 它们)。
func TestServerMgmtAPIReturnedFormat(t *testing.T) {
	msg := fmt.Sprintf(errMgmtAPIReturned, 404, "boom-body")
	assertChineseError(t, "errMgmtAPIReturned", msg)
	if !strings.Contains(msg, "404") || !strings.Contains(msg, "boom-body") {
		t.Fatalf("格式串必须保留状态码与正文: %q", msg)
	}
}

// TestServerMgmtSkillExistsSentinel 守卫跨栈契约：服务端的
// duplicate-skill 的文案必须带上 "已存在" 标记，
// web/src/app/(main)/system/skills/page.tsx 会用 (msg.includes("已存在"))
// 切到覆盖确认流程。这里一旦漂移，覆盖功能会静默失效。
func TestServerMgmtSkillExistsSentinel(t *testing.T) {
	msg := errMgmtSkillExistsPre + "my-skill" + errMgmtSkillExistsPost
	assertChineseError(t, "skillExists", msg)
	if !strings.Contains(msg, "已存在") {
		t.Fatalf("缺少前端镜像标记 '已存在': %q", msg)
	}
	if !strings.Contains(msg, "my-skill") {
		t.Fatalf("文案里必须包含 skill 名称: %q", msg)
	}
}

// TestServerMgmtPGGate503Localized 走每个管理处理器都会经过的
// 共享 DB 关卡：没有 PostgreSQL 句柄时，pg() 写出 503
// 并返回 nil。503 正文必须是中文。
func TestServerMgmtPGGate503Localized(t *testing.T) {
	s := &Server{m: &Manager{}} // pg == nil
	rec := httptest.NewRecorder()
	if got := s.pg(rec); got != nil {
		t.Fatal("没有 DB 句柄时 pg() 必须返回 nil")
	}
	if rec.Code != 503 {
		t.Fatalf("期望状态码 503，实际 %d", rec.Code)
	}
	assertChineseError(t, "pg.503", decodeErrorField(t, rec.Body.Bytes()))
}

// TestServerMgmtDeleteTaskBadIDLocalized 让 pgDeleteTask 走到第一个
// 守卫(非数字 id → 400)，不需要任何 DB 或引擎。
func TestServerMgmtDeleteTaskBadIDLocalized(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	s.pgDeleteTask(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，实际 %d (正文 %s)", rec.Code, rec.Body.Bytes())
	}
	got := decodeErrorField(t, rec.Body.Bytes())
	assertChineseError(t, "deleteTask.badID", got)
	if got != errMgmtTaskIDInvalid {
		t.Fatalf("必须与 errMgmtTaskIDInvalid 一致: %q", got)
	}
}

// TestServerMgmtListModelsNoKeyLocalized 在没有 api key 也没有
// profile id 的情况下驱动 pgListModels，因此它在触碰 DB 之前返回 {"ok":false,"error":...}。
func TestServerMgmtListModelsNoKeyLocalized(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/llm/models", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	s.pgListModels(rec, req)
	if rec.Code != 200 {
		t.Fatalf("期望状态码 200，实际 %d", rec.Code)
	}
	got := decodeErrorField(t, rec.Body.Bytes())
	assertChineseError(t, "listModels.noKey", got)
	if got != errMgmtNoAPIKey {
		t.Fatalf("必须与 errMgmtNoAPIKey 一致: %q", got)
	}
}

// TestServerMgmtValidateTemplateLocalized 走提示词模板编辑器
// 校验器(纯函数，不涉及 DB)：解析错误与
// 不允许的变量两个分支都必须返回中文。
func TestServerMgmtValidateTemplateLocalized(t *testing.T) {
	if msg := validateTemplate("{{", nil); msg == "" {
		t.Fatal("损坏的模板必须返回错误文案")
	} else {
		assertChineseError(t, "validateTemplate.syntax", msg)
	}
	msg := validateTemplate("{{.Bogus}}", nil)
	if msg == "" {
		t.Fatal("不在允许列表内的变量必须返回错误文案")
	}
	assertChineseError(t, "validateTemplate.var", msg)
	if !strings.Contains(msg, "Bogus") {
		t.Fatalf("文案里必须包含变量名: %q", msg)
	}
}

// TestServerMgmtGlobalPromptVarsLocalized 把模板变量目录的帮助
// 文案(渲染在提示词编辑器的变量面板里)固定为中文。
func TestServerMgmtGlobalPromptVarsLocalized(t *testing.T) {
	if len(globalPromptVars) == 0 {
		t.Fatal("globalPromptVars 为空")
	}
	for _, v := range globalPromptVars {
		assertChineseError(t, "globalPromptVar."+v.Name, v.Description)
	}
}
