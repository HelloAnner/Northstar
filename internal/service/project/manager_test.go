package project

import (
	"testing"

	"northstar/internal/service/calculator"
	"northstar/internal/service/store"
)

func TestListProjectsHasDataRequiresCompanies(t *testing.T) {
	dataDir := t.TempDir()
	memStore := store.NewMemoryStore()
	engine := calculator.NewEngine(memStore)

	manager, err := NewManager(dataDir, memStore, engine)
	if err != nil {
		t.Fatalf("create manager failed: %v", err)
	}

	projectSummary, err := manager.CreateProject("demo")
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	if err := manager.SaveNow(); err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	index := manager.ListProjects()
	if len(index.Items) != 1 {
		t.Fatalf("expected 1 project, got %d", len(index.Items))
	}

	got := index.Items[0]
	if got.ProjectID != projectSummary.ProjectID {
		t.Fatalf("unexpected project id: %s", got.ProjectID)
	}
	if got.HasData {
		t.Fatalf("expected hasData false when no companies, got true")
	}
	if got.CompanyCount != 0 {
		t.Fatalf("expected companyCount 0, got %d", got.CompanyCount)
	}
}
