package db

import (
	"testing"
	"unicode"
)

// TestManualTaskScopeSummaryLocalized 钉住手工新增任务范围的来源标签。
// 它存放在 task_asset_links.source_summary（AddTaskScope / SetTaskAssetSource），
// 并在任务详情的会话页与资产页原样渲染；一旦回退成谚文，同一个面板里就会出现
// 混语言的来源标签。task_assets_test.go 已经通过符号断言存储值等于这个常量，
// 这里再钉住常量本身，用户可见文案也就一并被保护。
func TestManualTaskScopeSummaryLocalized(t *testing.T) {
	if manualTaskScopeSummary == "" {
		t.Fatal("manualTaskScopeSummary: 空字符串")
	}
	hasHan := false
	for _, r := range manualTaskScopeSummary {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("manualTaskScopeSummary: 仍残留谚文: %q", manualTaskScopeSummary)
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	if !hasHan {
		t.Fatalf("manualTaskScopeSummary: 没有汉字: %q", manualTaskScopeSummary)
	}
}
