package v3

import "testing"

func TestBuildExportContentDisposition(t *testing.T) {
	t.Parallel()

	got := buildExportContentDisposition(2025, 12)
	want := "attachment; filename=\"monthly-report-2025-12.xlsx\"; filename*=UTF-8''2025%E5%B9%B412%E6%9C%88%E6%9C%88%E6%8A%A5.xlsx"
	if got != want {
		t.Fatalf("content-disposition mismatch:\n got: %s\nwant: %s", got, want)
	}
}

