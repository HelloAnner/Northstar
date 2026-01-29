package calculator

import "testing"

func TestDagRecomputeOnRetailChange(t *testing.T) {
	dag := NewDag()

	dag.SetRetailCurrent("c1", 100)
	dag.RecomputeFrom("c1.retailCurrent")

	if got, want := dag.GetTotalRetailCurrent(), 100.0; got != want {
		t.Fatalf("GetTotalRetailCurrent=%v, want %v", got, want)
	}

	dag.SetRetailCurrent("c2", 50)
	dag.RecomputeFrom("c2.retailCurrent")

	if got, want := dag.GetTotalRetailCurrent(), 150.0; got != want {
		t.Fatalf("GetTotalRetailCurrent=%v, want %v", got, want)
	}

	dag.SetRetailCurrent("c1", 80)
	dag.RecomputeFrom("c1.retailCurrent")

	if got, want := dag.GetTotalRetailCurrent(), 130.0; got != want {
		t.Fatalf("GetTotalRetailCurrent=%v, want %v", got, want)
	}
}
