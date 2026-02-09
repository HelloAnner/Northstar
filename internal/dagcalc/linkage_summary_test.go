/**
 * 汇总节点联动测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package dagcalc

import (
	"testing"

	"northstar/internal/model"
)

func TestDAG_IncludesSummaryNodes(t *testing.T) {
	index, err := LoadTemplateIndex()
	if err != nil {
		t.Fatalf("load template index: %v", err)
	}
	code := indexFirstCode(index, "批发")
	if code == "" {
		t.Fatalf("missing code")
	}
	wr := &model.WholesaleRetail{
		ID:           11,
		IndustryType: "wholesale",
		IndustryCode: code,
		RowNo:        1,
	}
	graph, err := BuildLinkageGraph(index, []*model.WholesaleRetail{wr}, nil)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	if !hasEdge(graph, "summary:micro_small:rate", "indicator:小微企业增速_当月") {
		t.Fatalf("missing micro_small summary edge")
	}
	if !hasEdge(graph, "summary:eat_wear_use:rate", "indicator:吃穿用增速_当月") {
		t.Fatalf("missing eat_wear_use summary edge")
	}
}

func hasEdge(g *Graph, from string, to string) bool {
	list := g.Edges[NodeID(from)]
	for _, n := range list {
		if n == NodeID(to) {
			return true
		}
	}
	return false
}
