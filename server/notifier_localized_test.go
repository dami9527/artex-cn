package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// TestNotifierDeliveryReasonsLocalized 是 F17 的守卫：notifier.go 写入
// notification_deliveries.last_error 的每一条投递失败/延后原因 —— 这些文案
// notify_api.go 会回显到 delivery-list.tsx 的「投递记录」表格 —— 必须是
// 简体中文(含汉字、不含谚文)。诊断性 log.Printf 保持
// 原文(Z2)，这里刻意不覆盖。
func TestNotifierDeliveryReasonsLocalized(t *testing.T) {
	// 用 ASCII 参数格式化每个常量，这样残留的谚文只可能
	// 来自常量本身，而不是测试数据注入的。
	cases := []struct {
		label string
		msg   string
	}{
		{"channelKindUnregistered", fmt.Sprintf(errDeliveryChannelKindUnregistered, "webhook")},
		{"snapshotUnrenderable", errDeliverySnapshotUnrenderable},
		{"channelLengthCapped", fmt.Sprintf(errDeliveryChannelLengthCapped, 3)},
		{"noDeliveredCount", fmt.Sprintf(errDeliveryNoDeliveredCount, 0)},
		{"retryExhausted", fmt.Sprintf(errDeliveryRetryExhausted, db.MaxNotifyAttempts, errors.New("boom"))},
		{"snapshotEmpty", fmt.Sprintf(errDeliverySnapshotEmpty, 7)},
		{"snapshotParse", fmt.Errorf(errDeliverySnapshotParse, 9, errors.New("unexpected end of JSON input")).Error()},
		{"batchAllUnparseable", fmt.Sprintf(errDeliveryBatchAllUnparseable, 2)},
	}
	for _, c := range cases {
		assertChineseError(t, c.label, c.msg)
	}
}

// TestNotifierParseSnapshotErrorsLocalized 走真实的 parseSnapshot 路径，
// 这些路径经 renderSingle → FailDeliveries 呈现为 last_error。parseSnapshot 是
// 包级纯函数，因此不需要 DB 或 Server。
func TestNotifierParseSnapshotErrorsLocalized(t *testing.T) {
	// 空快照。
	if _, err := parseSnapshot(&db.NotificationDelivery{ID: 7}); err == nil {
		t.Fatal("空快照不应无错通过")
	} else {
		assertChineseError(t, "parseSnapshot.empty", err.Error())
		if !strings.Contains(err.Error(), "7") {
			t.Fatalf("parseSnapshot.empty: 文案里没有投递条目 ID 7: %q", err.Error())
		}
	}
	// 损坏的 JSON 快照。
	if _, err := parseSnapshot(&db.NotificationDelivery{ID: 9, Snapshot: []byte("{bad")}); err == nil {
		t.Fatal("损坏的 JSON 快照不应无错通过")
	} else {
		assertChineseError(t, "parseSnapshot.malformed", err.Error())
	}
}

// TestNotifierRenderBatchAllUnparseableLocalized 驱动 renderBatch 真实的
// 「所有快照都无法解析」分支。快照全为空时，每条投递都在
// itemFor 之前(因而也在 n.pg 之前)被跳过，因此零值 Notifier 就能到达
// errDeliveryBatchAllUnparseable 返回，不触碰数据库。
func TestNotifierRenderBatchAllUnparseableLocalized(t *testing.T) {
	n := &Notifier{}
	deliveries := []*db.NotificationDelivery{{ID: 1}, {ID: 2}}
	if _, _, err := n.renderBatch(context.Background(), deliveries, "", 30); err == nil {
		t.Fatal("所有快照都无法解析时必须报错")
	} else {
		assertChineseError(t, "renderBatch.allUnparseable", err.Error())
	}
}
