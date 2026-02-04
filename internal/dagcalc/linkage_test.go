/**
 * 联动构图测试
 *
 * @author Anner
 * Created on 2026/2/4
 */

package dagcalc

import (
	"testing"

	"northstar/internal/model"
)

func TestBuildLinkageGraphImpact(t *testing.T) {
	index, err := LoadTemplateIndex()
	if err != nil {
		t.Fatalf("load template index: %v", err)
	}
	code := indexFirstCode(index, "批发")
	if code == "" {
		t.Fatalf("missing code")
	}
	wr := &model.WholesaleRetail{
		ID:           10,
		IndustryType: "wholesale",
		IndustryCode: code,
		RowNo:        1,
	}
	graph, err := BuildLinkageGraph(index, []*model.WholesaleRetail{wr}, nil)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	anchor := BuildNodeID("wr", wr.ID, "retailCurrentMonth")
	impact := ImpactRange(graph, anchor)
	if !containsImpactNode(impact, string(BuildNodeID("wr", wr.ID, "retailMonthRate"))) {
		t.Fatalf("impact missing retailMonthRate")
	}
	if !containsImpactNode(impact, "indicator:limitAbove_month_value") {
		t.Fatalf("impact missing limitAbove_month_value")
	}
}

func containsImpactNode(nodes []ImpactNode, id string) bool {
	for _, n := range nodes {
		if n.NodeID == id {
			return true
		}
	}
	return false
}

func indexFirstCode(index *TemplateIndex, sheet string) string {
	rows := index.codeRows[sheet]
	for code := range rows {
		if code != "" {
			return code
		}
	}
	return ""
}
