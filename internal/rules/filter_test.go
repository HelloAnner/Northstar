/**
 * 规则过滤测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import "testing"

func TestFilterByMode(t *testing.T) {
	rows := []CompanyRow{
		{RowID: 1, CompanyScale: 1, IsSmallMicro: false, CurrentValue: 6},
		{RowID: 2, CompanyScale: 3, IsSmallMicro: true, CurrentValue: -4},
	}

	positive := filterByMode(rows, "positive_current")
	if len(positive) != 1 || positive[0].RowID != 1 {
		t.Fatalf("unexpected positive rows: %+v", positive)
	}

	negative := filterByMode(rows, "negative_current")
	if len(negative) != 1 || negative[0].RowID != 2 {
		t.Fatalf("unexpected negative rows: %+v", negative)
	}

	largeScale := filterByMode(rows, "large_scale_only")
	if len(largeScale) != 1 || largeScale[0].RowID != 1 {
		t.Fatalf("unexpected large scale rows: %+v", largeScale)
	}

	excludeSmall := filterByMode(rows, "exclude_small_micro")
	if len(excludeSmall) != 1 || excludeSmall[0].RowID != 1 {
		t.Fatalf("unexpected exclude small rows: %+v", excludeSmall)
	}
}

func TestFilterByModeUnknownKeepsOriginalRows(t *testing.T) {
	rows := []CompanyRow{
		{RowID: 1, CurrentValue: 1},
		{RowID: 2, CurrentValue: -1},
	}

	filtered := filterByMode(rows, "unknown")
	if len(filtered) != len(rows) {
		t.Fatalf("unexpected row count: %d", len(filtered))
	}
	if filtered[0].RowID != rows[0].RowID || filtered[1].RowID != rows[1].RowID {
		t.Fatalf("unexpected filtered rows: %+v", filtered)
	}
}
