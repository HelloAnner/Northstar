package dagcalc

import "testing"

func TestImpactRangeIncludesParentsAndChildren(t *testing.T) {
	g := NewGraph()
	g.AddEdge("a", "b")
	g.AddReverseEdge("b", "a")
	nodes := ImpactRange(g, "b")
	if !containsNode(nodes, "a") {
		t.Fatalf("expected impact to include parent node")
	}
	if !containsNode(nodes, "b") {
		t.Fatalf("expected impact to include anchor node")
	}
}

func containsNode(nodes []ImpactNode, id string) bool {
	for _, node := range nodes {
		if node.NodeID == id {
			return true
		}
	}
	return false
}

func TestImpactRangeReturnsCoords(t *testing.T) {
	g := NewGraph()
	g.AddUICoord("x", UICoord{RowID: "wr:1", ColumnKey: "salesCurrentMonth"})
	nodes := ImpactRange(g, "x")
	if len(nodes) == 0 {
		t.Fatalf("expected nodes")
	}
	if nodes[0].UICoord == nil {
		t.Fatalf("expected ui coord")
	}
	if nodes[0].UICoord.RowID != "wr:1" {
		t.Fatalf("unexpected row id")
	}
}

func TestImpactRangeReturnsIndicatorID(t *testing.T) {
	g := NewGraph()
	g.AddIndicatorID("k", "limitAbove_month_value")
	nodes := ImpactRange(g, "k")
	if len(nodes) == 0 {
		t.Fatalf("expected nodes")
	}
	if nodes[0].IndicatorID != "limitAbove_month_value" {
		t.Fatalf("expected indicator id")
	}
}
