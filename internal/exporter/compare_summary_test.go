/**
 * 三方对比汇总测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package exporter

import "testing"

func TestBuildCompareSummary(t *testing.T) {
	summary := BuildCompareSummary(12, 8, nil)
	if len(summary.Items) != 3 {
		t.Fatalf("unexpected items: %d", len(summary.Items))
	}
	assertCompareItem(t, summary.Items[0], "raw", "pass")
	assertCompareItem(t, summary.Items[1], "business", "pass")
	assertCompareItem(t, summary.Items[2], "export", "pass")
}

func TestBuildCompareSummary_WithMissing(t *testing.T) {
	summary := BuildCompareSummary(0, 0, []string{"批发", "零售"})
	assertCompareItem(t, summary.Items[0], "raw", "warn")
	assertCompareItem(t, summary.Items[1], "business", "warn")
	assertCompareItem(t, summary.Items[2], "export", "warn")
}

func assertCompareItem(t *testing.T, item CompareItem, key, status string) {
	t.Helper()
	if item.Key != key {
		t.Fatalf("key mismatch: got=%s want=%s", item.Key, key)
	}
	if item.Status != status {
		t.Fatalf("status mismatch for %s: got=%s want=%s", key, item.Status, status)
	}
	if item.Message == "" {
		t.Fatalf("message empty for %s", key)
	}
}
