package parser

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWRParser_ParseSheet_MainSheets(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx")
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("open excel: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	p := NewWRParser(f)
	records, err := p.ParseSheet("批发")
	if err != nil {
		t.Fatalf("parse 批发: %v", err)
	}
	if len(records) == 0 {
		t.Fatalf("批发 parsed 0 records")
	}
	if records[0].DataYear != 2025 || records[0].DataMonth != 12 {
		t.Fatalf("unexpected ym: %d-%02d", records[0].DataYear, records[0].DataMonth)
	}
	if records[0].Name == "" || records[0].IndustryCode == "" {
		t.Fatalf("missing base fields: name=%q industry=%q", records[0].Name, records[0].IndustryCode)
	}
}

