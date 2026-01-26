package store

import (
	"sync"
	"testing"

	"northstar/internal/model"
)

// TestNewMemoryStore 测试创建存储
func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore() returned nil")
	}
	if store.Count() != 0 {
		t.Errorf("New store should be empty, got %d companies", store.Count())
	}
}

// TestAddCompany 测试添加企业
func TestAddCompany(t *testing.T) {
	store := NewMemoryStore()

	company := &model.Company{
		ID:                 "test-1",
		Name:               "测试企业",
		RetailCurrentMonth: 1000,
	}

	store.AddCompany(company)

	if store.Count() != 1 {
		t.Errorf("Store should have 1 company, got %d", store.Count())
	}

	retrieved, err := store.GetCompany("test-1")
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}
	if retrieved.Name != "测试企业" {
		t.Errorf("Company name = %s, want 测试企业", retrieved.Name)
	}
}

// TestGetCompanyNotFound 测试获取不存在的企业
func TestGetCompanyNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.GetCompany("non-existent")
	if err == nil {
		t.Error("GetCompany should return error for non-existent company")
	}
}

// TestSetCompanies 测试批量设置企业
func TestSetCompanies(t *testing.T) {
	store := NewMemoryStore()

	companies := []*model.Company{
		{ID: "c1", Name: "企业1", RetailCurrentMonth: 100},
		{ID: "c2", Name: "企业2", RetailCurrentMonth: 200},
		{ID: "c3", Name: "企业3", RetailCurrentMonth: 300},
	}

	store.SetCompanies(companies)

	if store.Count() != 3 {
		t.Errorf("Store should have 3 companies, got %d", store.Count())
	}

	// 验证原始值保存
	c, _ := store.GetCompany("c1")
	if c.OriginalRetailCurrentMonth != 100 {
		t.Errorf("OriginalRetailCurrentMonth = %v, want 100", c.OriginalRetailCurrentMonth)
	}
}

// TestUpdateCompanyRetail 测试更新企业零售额
func TestUpdateCompanyRetail(t *testing.T) {
	store := NewMemoryStore()

	company := &model.Company{
		ID:                      "test-1",
		RetailCurrentMonth:      1000,
		RetailCurrentCumulative: 10000,
	}
	store.AddCompany(company)

	// 更新零售额
	updated, err := store.UpdateCompanyRetail("test-1", 1200)
	if err != nil {
		t.Fatalf("UpdateCompanyRetail failed: %v", err)
	}

	// 验证当月值更新
	if updated.RetailCurrentMonth != 1200 {
		t.Errorf("RetailCurrentMonth = %v, want 1200", updated.RetailCurrentMonth)
	}

	// 验证累计值同步更新
	expectedCumulative := 10000.0 + (1200.0 - 1000.0)
	if updated.RetailCurrentCumulative != expectedCumulative {
		t.Errorf("RetailCurrentCumulative = %v, want %v", updated.RetailCurrentCumulative, expectedCumulative)
	}
}

// TestResetCompanies 测试重置企业数据
func TestResetCompanies(t *testing.T) {
	store := NewMemoryStore()

	companies := []*model.Company{
		{ID: "c1", RetailCurrentMonth: 100, RetailCurrentCumulative: 1000},
		{ID: "c2", RetailCurrentMonth: 200, RetailCurrentCumulative: 2000},
	}
	store.SetCompanies(companies)

	// 修改数据
	store.UpdateCompanyRetail("c1", 150)
	store.UpdateCompanyRetail("c2", 250)

	// 重置全部
	store.ResetCompanies(nil)

	c1, _ := store.GetCompany("c1")
	if c1.RetailCurrentMonth != 100 {
		t.Errorf("After reset, c1 RetailCurrentMonth = %v, want 100", c1.RetailCurrentMonth)
	}

	c2, _ := store.GetCompany("c2")
	if c2.RetailCurrentMonth != 200 {
		t.Errorf("After reset, c2 RetailCurrentMonth = %v, want 200", c2.RetailCurrentMonth)
	}
}

// TestResetSpecificCompanies 测试重置指定企业
func TestResetSpecificCompanies(t *testing.T) {
	store := NewMemoryStore()

	companies := []*model.Company{
		{ID: "c1", RetailCurrentMonth: 100},
		{ID: "c2", RetailCurrentMonth: 200},
	}
	store.SetCompanies(companies)

	store.UpdateCompanyRetail("c1", 150)
	store.UpdateCompanyRetail("c2", 250)

	// 只重置 c1
	store.ResetCompanies([]string{"c1"})

	c1, _ := store.GetCompany("c1")
	if c1.RetailCurrentMonth != 100 {
		t.Errorf("After reset, c1 should be 100, got %v", c1.RetailCurrentMonth)
	}

	c2, _ := store.GetCompany("c2")
	if c2.RetailCurrentMonth != 250 {
		t.Errorf("c2 should not be reset, got %v", c2.RetailCurrentMonth)
	}
}

// TestConcurrentAccess 测试并发访问安全性
func TestConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()

	companies := make([]*model.Company, 100)
	for i := 0; i < 100; i++ {
		companies[i] = &model.Company{
			ID:                 string(rune('A' + i)),
			RetailCurrentMonth: float64(i * 100),
		}
	}
	store.SetCompanies(companies)

	var wg sync.WaitGroup

	// 并发读取
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.GetAllCompanies()
		}()
	}

	// 并发写入
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			store.UpdateCompanyRetail(string(rune('A'+idx)), float64(idx*200))
		}(i)
	}

	wg.Wait()

	// 验证没有 panic，数据一致
	if store.Count() != 100 {
		t.Errorf("After concurrent access, count = %d, want 100", store.Count())
	}
}

// TestClear 测试清空数据
func TestClear(t *testing.T) {
	store := NewMemoryStore()

	store.AddCompany(&model.Company{ID: "c1"})
	store.AddCompany(&model.Company{ID: "c2"})

	if store.Count() != 2 {
		t.Fatalf("Before clear, count should be 2")
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("After clear, count should be 0, got %d", store.Count())
	}
}

// TestConfig 测试配置管理
func TestConfig(t *testing.T) {
	store := NewMemoryStore()

	config := store.GetConfig()
	if config.CurrentMonth != 6 {
		t.Errorf("Default CurrentMonth = %d, want 6", config.CurrentMonth)
	}

	store.UpdateConfig(map[string]interface{}{
		"currentMonth": 9,
	})

	config = store.GetConfig()
	if config.CurrentMonth != 9 {
		t.Errorf("Updated CurrentMonth = %d, want 9", config.CurrentMonth)
	}
}
