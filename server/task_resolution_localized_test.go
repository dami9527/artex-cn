package server

import (
	"testing"

	"github.com/Autumn-27/artex/db"
)

// TestTaskLLMResolutionLocalized 守护 F18：`GET /api/tasks/{id}/llm/resolution` 返回的
// 每一条面向用户的 reason/label（在任务 LLM 设置链、system/llm 界面通过
// resolutionLabel、以及链编辑器里渲染）都必须是简体中文——含汉字、不含谚文。
// 来源枚举值与英文 fmt.Errorf 包装不在本次范围内，不在此断言。
func TestTaskLLMResolutionLocalized(t *testing.T) {
	// 1) 每个本地化常量都是中文。任何一个改回别的语言都会在这里失败。
	for _, c := range []struct{ label, msg string }{
		{"reasonLLMProfileMissing", reasonLLMProfileMissing},
		{"reasonLLMProfileNoAPIKey", reasonLLMProfileNoAPIKey},
		{"reasonLLMProfileInvalid", reasonLLMProfileInvalid},
		{"reasonTaskLLMChainExhausted", reasonTaskLLMChainExhausted},
		{"reasonNoLLMAvailable", reasonNoLLMAvailable},
		{"sourceNameGlobalConfig", sourceNameGlobalConfig},
	} {
		assertChineseError(t, c.label, c.msg)
	}

	// 2) 钉住两条不需要数据库就能走到的原因。resolutionFromProfile 在 profile 为 nil
	// 或 API key 为空时会在接触 providerForProfile 之前返回，因此零值 Server 就能
	// 走到真实代码路径。
	s := &Server{}

	missing := s.resolutionFromProfile(nil, "task_chain")
	if missing.Available {
		t.Fatalf("nil profile must be unavailable, got %+v", missing)
	}
	if missing.Reason != reasonLLMProfileMissing {
		t.Fatalf("nil profile reason = %q, want %q", missing.Reason, reasonLLMProfileMissing)
	}
	assertChineseError(t, "resolutionFromProfile(nil).Reason", missing.Reason)

	noKey := s.resolutionFromProfile(&db.LLMProfile{Name: "p", Format: "openai", Model: "m"}, "task_chain")
	if noKey.Available {
		t.Fatalf("blank API key must be unavailable, got %+v", noKey)
	}
	if noKey.Reason != reasonLLMProfileNoAPIKey {
		t.Fatalf("blank-key reason = %q, want %q", noKey.Reason, reasonLLMProfileNoAPIKey)
	}
	assertChineseError(t, "resolutionFromProfile(noKey).Reason", noKey.Reason)
}
