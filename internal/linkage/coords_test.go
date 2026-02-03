/**
 * 联动坐标映射测试
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import (
	"strings"
	"testing"

	"northstar/internal/model"
)

func TestBuildGraphExcelCoords(t *testing.T) {
	index, err := loadTemplateIndex()
	if err != nil {
		t.Fatalf("load template index: %v", err)
	}

	wrCode := pickFirstCode(index, "批发")
	acCode := pickFirstCode(index, "住宿")
	if wrCode == "" || acCode == "" {
		t.Fatalf("template missing codes: wr=%q ac=%q", wrCode, acCode)
	}

	wr := &model.WholesaleRetail{
		ID:           1,
		IndustryType: "wholesale",
		IndustryCode: wrCode,
		RowNo:        1,
	}
	ac := &model.AccommodationCatering{
		ID:           2,
		IndustryType: "accommodation",
		IndustryCode: acCode,
		RowNo:        1,
	}

	graph, err := BuildGraph(BuildGraphOptions{
		WRRecords:     []*model.WholesaleRetail{wr},
		ACRecords:     []*model.AccommodationCatering{ac},
		TemplateIndex: index,
	})
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	wrNode := BuildNodeID("wr", wr.ID, "salesCurrentMonth")
	wrCoords := graph.ExcelCoords[wrNode]
	if !hasSheetColumn(wrCoords, "批发", "D") || !hasSheetColumn(wrCoords, "批零总表", "D") {
		t.Fatalf("wr coords missing sheets: %v", wrCoords)
	}

	acNode := BuildNodeID("ac", ac.ID, "salesCurrentMonth")
	acCoords := graph.ExcelCoords[acNode]
	if !hasSheetColumn(acCoords, "住宿", "D") || !hasSheetColumn(acCoords, "住餐总表", "D") {
		t.Fatalf("ac coords missing sheets: %v", acCoords)
	}
}

func hasSheetColumn(coords []ExcelCoord, sheet string, col string) bool {
	for _, c := range coords {
		if c.Sheet == sheet && strings.HasPrefix(c.Cell, col) {
			return true
		}
	}
	return false
}
