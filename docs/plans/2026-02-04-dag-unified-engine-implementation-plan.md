# Unified DAG Engine Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Introduce a new `internal/dagcalc` package that unifies DAG graph, impact range, forward/reverse compute entrypoints, and migrate linkage preview to use it.

**Architecture:** Build a standalone DAG engine with graph + rules + engine APIs. First deliver the shared graph/impact logic, then adapt linkage preview to the new package, then extend to forward/reverse compute in incremental steps.

**Tech Stack:** Go (internal packages), existing store/calculator/linkage modules.

---

### Task 1: Create dagcalc graph + impact API (skeleton)

**Files:**
- Create: `internal/dagcalc/types.go`
- Create: `internal/dagcalc/graph.go`
- Create: `internal/dagcalc/impact.go`
- Test: `internal/dagcalc/impact_test.go`

**Step 1: Write the failing test**
```go
func TestImpactRangeIncludesParentsAndChildren(t *testing.T) {
    g := NewGraph()
    g.AddEdge("a", "b")
    g.AddReverseEdge("b", "a")
    nodes := ImpactRange(g, "b")
    if !contains(nodes, "a") || !contains(nodes, "b") {
        t.Fatalf("expected impact to include both parent and child")
    }
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/dagcalc -run TestImpactRangeIncludesParentsAndChildren`
Expected: FAIL (undefined functions/types)

**Step 3: Write minimal implementation**
- Add `Graph` with `Edges`/`ReverseEdges` maps and `AddEdge`/`AddReverseEdge`.
- Add `ImpactRange` BFS over both directions.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/dagcalc -run TestImpactRangeIncludesParentsAndChildren`
Expected: PASS

---

### Task 2: Add node metadata + coordinates support

**Files:**
- Modify: `internal/dagcalc/types.go`
- Modify: `internal/dagcalc/graph.go`
- Test: `internal/dagcalc/impact_test.go`

**Step 1: Write the failing test**
```go
func TestImpactRangeReturnsCoords(t *testing.T) {
    g := NewGraph()
    g.AddNodeCoord("x", UICoord{RowID: "wr:1", ColumnKey: "salesCurrentMonth"})
    nodes := ImpactRange(g, "x")
    if nodes[0].UICoord == nil {
        t.Fatalf("expected ui coord")
    }
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/dagcalc -run TestImpactRangeReturnsCoords`
Expected: FAIL (missing coord types)

**Step 3: Write minimal implementation**
- Add `UICoord`, `ExcelCoord`, `ImpactNode`.
- Support `AddNodeCoord` and include coords in `ImpactRange` output.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/dagcalc -run TestImpactRangeReturnsCoords`
Expected: PASS

---

### Task 3: Migrate linkage preview to dagcalc

**Files:**
- Modify: `internal/linkage/impact.go`
- Modify: `internal/linkage/build.go`
- Modify: `internal/linkage/types.go`
- Test: `internal/linkage/impact_test.go`

**Step 1: Write the failing test**
- Extend existing linkage test to assert the new dagcalc path returns the same nodes.

**Step 2: Run test to verify it fails**
Run: `go test ./internal/linkage -run TestComputeImpact`
Expected: FAIL after refactor placeholder

**Step 3: Write minimal implementation**
- Build `dagcalc.Graph` from existing linkage build logic.
- Replace `ComputeImpact` with `dagcalc.ImpactRange`.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/linkage -run TestComputeImpact`
Expected: PASS

---

### Task 4: Introduce forward/reverse engine interfaces (no full compute yet)

**Files:**
- Create: `internal/dagcalc/engine.go`
- Test: `internal/dagcalc/engine_test.go`

**Step 1: Write the failing test**
```go
func TestEngineForwardRecalcReturnsPlan(t *testing.T) {
    eng := NewEngine(NewGraph())
    plan := eng.ForwardRecalc("a")
    if plan == nil {
        t.Fatalf("expected plan")
    }
}
```

**Step 2: Run test to verify it fails**
Run: `go test ./internal/dagcalc -run TestEngineForwardRecalcReturnsPlan`
Expected: FAIL (missing engine)

**Step 3: Write minimal implementation**
- Add `Plan` struct and `ForwardRecalc`/`ReverseAdjust` stubs.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/dagcalc -run TestEngineForwardRecalcReturnsPlan`
Expected: PASS

---

### Task 5: Wire API entrypoints to use dagcalc impact

**Files:**
- Modify: `internal/api/v3/linkage_preview.go`
- Test: `internal/api/v3/linkage_preview_test.go`

**Step 1: Write the failing test**
- Ensure preview still returns identical nodes after engine swap.

**Step 2: Run test to verify it fails**
Run: `go test ./internal/api/v3 -run TestPreviewLinkage`
Expected: FAIL if wiring missing

**Step 3: Write minimal implementation**
- Use `dagcalc.ImpactRange` output.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/api/v3 -run TestPreviewLinkage`
Expected: PASS

