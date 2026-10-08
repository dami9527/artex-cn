package server

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestChatMentionErrorsLocalized 是 F3b chat_mentions.go 的守卫：每一条用户可见
// @提及(引用)的错误响应必须是简体中文(含汉字、不含谚文)。下面
// 线上标记标签(chatMentionPattern·chatMentionKinds)、智能体输入
// 快照表头，以及写入该快照的截断标记，按设计保持
// 原文，这里刻意不检查。
func TestChatMentionErrorsLocalized(t *testing.T) {
	// 固定的用户可见字面量。
	for label, msg := range map[string]string{
		"badID":       errChatMentionBadID,
		"tooMany":     errChatMentionTooMany,
		"badSearch":   errChatMentionBadSearch,
		"dataUnavail": errChatMentionDataUnavail,
		"tooLarge":    errChatMentionTooLarge,
	} {
		assertChineseError(t, label, msg)
	}

	// parseChatMentions —— 合法线上标记里的非正数 id。
	if _, err := parseChatMentions("@[漏洞#0]"); err == nil {
		t.Fatal("非法引用 ID 不应通过")
	} else {
		assertChineseError(t, "parse.badID", err.Error())
	}

	// parseChatMentions —— 超过单条消息的引用上限(11 个不同 id)。
	var b strings.Builder
	for i := 1; i <= 11; i++ {
		fmt.Fprintf(&b, "@[漏洞#%d] ", i)
	}
	if _, err := parseChatMentions(b.String()); err == nil {
		t.Fatal("引用数超限不应通过")
	} else {
		assertChineseError(t, "parse.tooMany", err.Error())
	}

	// composeChatMentionMessage —— 有引用但数据库为 nil。
	if _, err := composeChatMentionMessage(nil, "@[漏洞#1]"); err == nil {
		t.Fatal("无 DB 时引用解析不应通过")
	} else {
		assertChineseError(t, "compose.dataUnavail", err.Error())
	}

	// searchChatMentions —— 非法 kind 与超长查询词在触碰数据库句柄
	// 之前就被拒绝，因此 &Server{} 就足以到达该分支。
	for _, q := range []string{"kind=unsupported", "q=" + url.QueryEscape(strings.Repeat("字", 201))} {
		w := httptest.NewRecorder()
		(&Server{}).searchChatMentions(w, httptest.NewRequest("GET", "/api/chat/mentions?"+q, nil))
		if w.Code != 400 {
			t.Fatalf("期望校验失败(400)，实际返回 %d (%s)", w.Code, q)
		}
		assertChineseError(t, "search."+q, decodeErrorField(t, w.Body.Bytes()))
	}

	// 展示标签映射与 UI 的引用类型一致，未找到模板
	// 对已知类型会解析成完整的中文文案。
	for _, kind := range []string{"finding", "asset", "company", "endpoint", "ip", "app", "root_domain", "subdomain", "service"} {
		if chatMentionKindLabel[kind] == "" {
			t.Fatalf("缺少类型标签: %s", kind)
		}
	}
	assertChineseError(t, "notFound", fmt.Sprintf(errChatMentionNotFoundFmt, chatMentionKindLabel["finding"], 7))
}
