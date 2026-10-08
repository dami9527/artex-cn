package notify

import (
	"strings"
	"testing"
)

// hasHan 判断字符串中是否含有 CJK 统一汉字。
// 简体中文文案必须带汉字，缺了汉字就说明本地化回退了。
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// hasHangul 判断字符串中是否含有谚文音节。
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

// assertChinese 断言文案是简体中文：必须含汉字，且不含任何谚文。
func assertChinese(t *testing.T, where, got string) {
	t.Helper()
	if !hasHan(got) {
		t.Errorf("%s: 未包含汉字: %q", where, got)
	}
	if hasHangul(got) {
		t.Errorf("%s: 仍残留谚文: %q", where, got)
	}
}

// TestSeverityLabelLocalized 检查所有严重级别枚举都输出简体中文标签。
// 标签由全部通知渠道(telegram·email·html·markdown·webhook·feishu·dingtalk·wecom)
// 共享，任何一个回退都会让全渠道消息混杂。
func TestSeverityLabelLocalized(t *testing.T) {
	// 与 UI status.severity 对齐：严重/高危/中危/低危。
	want := map[string]string{
		"critical": "严重",
		"high":     "高危",
		"medium":   "中危",
		"low":      "低危",
	}
	for sev, label := range want {
		got := SeverityLabel(sev)
		assertChinese(t, "SeverityLabel("+sev+")", got)
		if !strings.Contains(got, label) {
			t.Errorf("SeverityLabel(%q)=%q, 必须包含 %q", sev, got, label)
		}
	}
	// 未知严重级别原样回显（不臆造不存在的值）。
	if got := SeverityLabel("made_up"); got != "made_up" {
		t.Errorf("未知严重级别必须原样保留，实际得到 %q", got)
	}
}

// TestStatusLabelLocalized 检查 9 种处置状态全部输出简体中文（UI status.finding
// 命名空间 B4a 与时一致），否则状态变更通知与界面会对不上。
func TestStatusLabelLocalized(t *testing.T) {
	want := map[string]string{
		"pending":        "待处理",
		"in_progress":    "处理中",
		"confirmed":      "已确认",
		"resolved":       "已处理",
		"fixed":          "已修复",
		"false_positive": "误报",
		"ignored":        "忽略",
		"duplicate":      "重复",
		"risk_accepted":  "风险接受",
	}
	for status, label := range want {
		got := StatusLabel(status)
		assertChinese(t, "StatusLabel("+status+")", got)
		if got != label {
			t.Errorf("StatusLabel(%q)=%q, 期望 %q", status, got, label)
		}
	}
	// 未知状态原样回显（不臆造不存在的标签）。
	if got := StatusLabel("weird_status"); got != "weird_status" {
		t.Errorf("未知状态必须原样保留，实际得到 %q", got)
	}
}

// TestItemTitlePlaceholderLocalized 检查名称与类型都为空的条目，其兜底标题是
// 中文占位符（未命名漏洞）。绝不输出空标题。
func TestItemTitlePlaceholderLocalized(t *testing.T) {
	got := Item{}.Title()
	assertChinese(t, "Item{}.Title()", got)
	// 有名称优先用名称，只有类型则用类型（兜底逻辑未回退）。
	if n := (Item{Name: "登录 SQLi"}).Title(); n != "登录 SQLi" {
		t.Errorf("名称优先逻辑被破坏，实际得到 %q", n)
	}
	if v := (Item{VulnClass: "XSS"}).Title(); v != "XSS" {
		t.Errorf("类型兜底逻辑被破坏，实际得到 %q", v)
	}
}

// TestAssetLineLocalized 检查资产列表的省略标注为中文，且不含谚文。
func TestAssetLineLocalized(t *testing.T) {
	// 未超上限：全部列出，分隔符与 assetLine 当前实现一致（中文顿号）。
	if got := assetLine([]string{"a.example.com", "b.example.com"}, 3); got != "a.example.com、b.example.com" {
		t.Errorf("分隔符必须与源码实现一致，实际得到 %q", got)
	}
	// 超过上限：前 limit 个 + 中文 "等 N 个"（N 为总数）。
	got := assetLine([]string{"a", "b", "c", "d", "e"}, 2)
	if hasHangul(got) {
		t.Errorf("资产列表仍残留谚文: %q", got)
	}
	if !strings.Contains(got, "等 5 个") {
		t.Errorf("总数 5 必须标注为 '等 5 个'，实际得到 %q", got)
	}
}
