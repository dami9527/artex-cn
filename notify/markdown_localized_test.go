package notify

import (
	"strings"
	"testing"
)

// koFreeItems 只构造不含谚文的条目（ASCII 的标题、类型、摘要）。
// 标签回退成谚文时整份渲染结果都会出现谚文，因此只有数据本身谚文为 0 时，
// 「整份输出谚文为 0」这条断言才能精确抓到标签回退。
func koFreeItems(n int) []Item {
	items := make([]Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, Item{
			FindingID: int64(i + 1),
			Name:      "login-flaw",
			VulnClass: "SQLi",
			Severity:  "high",
			Summary:   "SQL injection via q param",
			Assets:    []string{"a.example.com"},
		})
	}
	return items
}

// TestMarkdownTitleLocalized 检查 markdownTitle 是否为简体中文。dingtalk·wecom
// 复用 markdown 渲染，feishu·html·telegram·webhook 也共用这个标题函数，
// 任何一个回退都会让这五个渠道的标题一起混杂。
func TestMarkdownTitleLocalized(t *testing.T) {
	// 多条（汇总）："漏洞汇总 · 共 N 条"
	got := markdownTitle(Message{Batch: true, Items: koFreeItems(3)})
	assertChinese(t, "markdownTitle(batch)", got)
	if !strings.Contains(got, "漏洞汇总") || !strings.Contains(got, "共 3 条") {
		t.Errorf("多条标题必须是 '漏洞汇总 · 共 3 条' 形式，实际得到 %q", got)
	}
	// 无条目："漏洞通知"
	empty := markdownTitle(Message{})
	assertChinese(t, "markdownTitle(empty)", empty)
	if empty != "漏洞通知" {
		t.Errorf("空消息标题必须是 '漏洞通知'，实际得到 %q", empty)
	}
}

// TestMarkdownBatchIntroLocalized 检查汇总导读（时间窗、条数、超量提示）是否为
// 简体中文。条目数据谚文为 0，所以输出里一出现谚文就是导读文案回退了。
func TestMarkdownBatchIntroLocalized(t *testing.T) {
	items := koFreeItems(3)

	// 有时间窗："近 N 分钟新增 N 个漏洞"
	win := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 3)
	if hasHangul(win) {
		t.Errorf("时间窗导读仍残留谚文: %q", win)
	}
	if !strings.Contains(win, "近 30 分钟") || !strings.Contains(win, "新增 3 个漏洞") {
		t.Errorf("时间窗导读必须是 '近 30 分钟新增 3 个漏洞' 形式，实际得到 %q", win)
	}

	// 无时间窗："新增 N 个漏洞"（不带分钟表述）
	noWin := markdownBatchIntro(Message{Batch: true}, items, 3)
	if hasHangul(noWin) {
		t.Errorf("导读仍残留谚文: %q", noWin)
	}
	if !strings.Contains(noWin, "新增 3 个漏洞") || strings.Contains(noWin, "分钟") {
		t.Errorf("无时间窗导读必须是 '新增 3 个漏洞'（不带分钟表述），实际得到 %q", noWin)
	}

	// 只装入一部分时："（本条显示前 N 条，其余 N 条将在下一条消息继续）"
	trunc := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 5)
	if hasHangul(trunc) {
		t.Errorf("超量提示仍残留谚文: %q", trunc)
	}
	for _, want := range []string{"前 3 条", "其余 2 条", "下一条消息"} {
		if !strings.Contains(trunc, want) {
			t.Errorf("超量提示必须包含 %q，实际得到 %q", want, trunc)
		}
	}
}

// TestWriteItemLabelsLocalized 检查单条详情渲染（markdown 三渠道共用路径）的
// 字段标签（状态变更、类型、资产、摘要）与详情链接是否为简体中文。
func TestWriteItemLabelsLocalized(t *testing.T) {
	it := Item{
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/1",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
	var b strings.Builder
	writeItem(&b, it, "", true)
	got := b.String()

	assertChinese(t, "writeItem(single)", got)
	for _, want := range []string{
		"**状态变更**：待处理 → 已修复",
		"**类型**：SQLi",
		"**资产**：a.example.com",
		"**摘要**：SQL injection via q param",
		"[查看详情](https://platform.example/finding/1)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("单条渲染必须包含 %q:\n%s", want, got)
		}
	}
}

// TestMarkdownBodyFooterLocalized 检查多条正文末尾的平台入口链接是否为简体中文
// （仅在 HomeURL 非空时追加）。
func TestMarkdownBodyFooterLocalized(t *testing.T) {
	m := Message{Batch: true, HomeURL: "https://platform.example", Items: koFreeItems(2)}
	body, kept := markdownBody(m, 0)
	if kept != 2 {
		t.Fatalf("不设上限(0)时必须装入全部 2 条，实际得到 %d", kept)
	}
	if !strings.Contains(body, "[在平台中查看全部](https://platform.example)") {
		t.Errorf("正文末尾必须有 '在平台中查看全部' 链接:\n%s", body)
	}
}
