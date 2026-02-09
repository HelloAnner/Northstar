/**
 * 联动影响范围测试
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import (
	"testing"

	"northstar/internal/model"
)

func TestImpactFromRetailCurrentMonth(t *testing.T) {
	index, err := loadTemplateIndex()
	if err != nil {
		t.Fatalf("load template index: %v", err)
	}
	wrCode := pickFirstCode(index, "批发")
	if wrCode == "" {
		t.Fatalf("missing wr code")
	}

	wr := &model.WholesaleRetail{
		ID:           10,
		IndustryType: "wholesale",
		IndustryCode: wrCode,
		RowNo:        1,
	}

	graph, err := BuildGraph(BuildGraphOptions{
		WRRecords:     []*model.WholesaleRetail{wr},
		ACRecords:     []*model.AccommodationCatering{},
		TemplateIndex: index,
	})
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	anchor := BuildNodeID("wr", wr.ID, "retailCurrentMonth")
	impact := ComputeImpact(graph, anchor)
	if !containsNode(impact, string(BuildNodeID("wr", wr.ID, "retailMonthRate"))) {
		t.Fatalf("impact missing retailMonthRate: %v", impact)
	}
	if !containsNode(impact, string(BuildNodeID("wr", wr.ID, "retailYoYDiff"))) {
		t.Fatalf("impact missing retailYoYDiff: %v", impact)
	}
	if !containsNode(impact, "indicator:限上社零额_当月值") {
		t.Fatalf("impact missing 限上社零额_当月值: %v", impact)
	}
}

func containsNode(nodes []ImpactNode, id string) bool {
	for _, n := range nodes {
		if n.NodeID == id {
			return true
		}
	}
	return false
}
