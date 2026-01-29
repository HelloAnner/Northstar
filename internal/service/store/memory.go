package store

import (
	"errors"
	"sync"

	"northstar/internal/model"
)

// MemoryStore 内存数据存储
type MemoryStore struct {
	companies map[string]*model.Company
	config    *model.Config
	mu        sync.RWMutex
}

func cloneCompany(c *model.Company) *model.Company {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

func cloneConfig(c *model.Config) *model.Config {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		companies: make(map[string]*model.Company),
		config: &model.Config{
			CurrentMonth:                 6,
			LastYearLimitBelowCumulative: 50000,
		},
	}
}

// GetAllCompanies 获取所有企业
func (s *MemoryStore) GetAllCompanies() []*model.Company {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*model.Company, 0, len(s.companies))
	for _, c := range s.companies {
		result = append(result, cloneCompany(c))
	}
	return result
}

// GetCompany 获取单个企业
func (s *MemoryStore) GetCompany(id string) (*model.Company, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}
	return cloneCompany(company), nil
}

// SetCompanies 设置企业列表
func (s *MemoryStore) SetCompanies(companies []*model.Company) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.companies = make(map[string]*model.Company)
	for _, c := range companies {
		// 兼容旧数据：若未持久化原始快照，则以当前 state 作为基线初始化一次。
		if !c.OriginalInitialized {
			c.OriginalInitialized = true
			c.OriginalRowNo = c.RowNo
			c.OriginalName = c.Name
			c.OriginalRetailLastYearMonth = c.RetailLastYearMonth
			c.OriginalRetailCurrentMonth = c.RetailCurrentMonth
			c.OriginalRetailLastYearCumulative = c.RetailLastYearCumulative
			c.OriginalRetailCurrentCumulative = c.RetailCurrentCumulative
			c.OriginalSalesLastYearMonth = c.SalesLastYearMonth
			c.OriginalSalesCurrentMonth = c.SalesCurrentMonth
			c.OriginalSalesLastYearCumulative = c.SalesLastYearCumulative
			c.OriginalSalesCurrentCumulative = c.SalesCurrentCumulative

			c.OriginalRoomRevenueLastYearMonth = c.RoomRevenueLastYearMonth
			c.OriginalRoomRevenueCurrentMonth = c.RoomRevenueCurrentMonth
			c.OriginalRoomRevenueLastYearCumulative = c.RoomRevenueLastYearCumulative
			c.OriginalRoomRevenueCurrentCumulative = c.RoomRevenueCurrentCumulative

			c.OriginalFoodRevenueLastYearMonth = c.FoodRevenueLastYearMonth
			c.OriginalFoodRevenueCurrentMonth = c.FoodRevenueCurrentMonth
			c.OriginalFoodRevenueLastYearCumulative = c.FoodRevenueLastYearCumulative
			c.OriginalFoodRevenueCurrentCumulative = c.FoodRevenueCurrentCumulative

			c.OriginalGoodsSalesLastYearMonth = c.GoodsSalesLastYearMonth
			c.OriginalGoodsSalesCurrentMonth = c.GoodsSalesCurrentMonth
			c.OriginalGoodsSalesLastYearCumulative = c.GoodsSalesLastYearCumulative
			c.OriginalGoodsSalesCurrentCumulative = c.GoodsSalesCurrentCumulative
		}
		s.companies[c.ID] = c
	}
}

// UpdateCompanyName 更新企业名称
func (s *MemoryStore) UpdateCompanyName(id string, name string) (*model.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}

	company.Name = name
	return cloneCompany(company), nil
}

// UpdateCompanyRetail 更新企业零售额
func (s *MemoryStore) UpdateCompanyRetail(id string, newRetailCurrentMonth float64) (*model.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}

	// 计算变化量，同步更新累计值
	delta := newRetailCurrentMonth - company.RetailCurrentMonth
	company.RetailCurrentMonth = newRetailCurrentMonth
	company.RetailCurrentCumulative += delta

	return cloneCompany(company), nil
}

// UpdateCompanyRetailLastYearMonth 更新企业上年同期零售额
func (s *MemoryStore) UpdateCompanyRetailLastYearMonth(id string, newRetailLastYearMonth float64) (*model.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}

	delta := newRetailLastYearMonth - company.RetailLastYearMonth
	company.RetailLastYearMonth = newRetailLastYearMonth
	company.RetailLastYearCumulative += delta

	return cloneCompany(company), nil
}

// UpdateCompanySales 更新企业销售额（本期）
func (s *MemoryStore) UpdateCompanySales(id string, newSalesCurrentMonth float64) (*model.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}

	delta := newSalesCurrentMonth - company.SalesCurrentMonth
	company.SalesCurrentMonth = newSalesCurrentMonth
	company.SalesCurrentCumulative += delta

	return cloneCompany(company), nil
}

// BatchUpdateCompanyRetail 批量更新企业零售额
func (s *MemoryStore) BatchUpdateCompanyRetail(updates map[string]float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, newValue := range updates {
		company, ok := s.companies[id]
		if !ok {
			continue
		}
		delta := newValue - company.RetailCurrentMonth
		company.RetailCurrentMonth = newValue
		company.RetailCurrentCumulative += delta
	}

	return nil
}

// UpdateCompanyRetailCumulative 更新企业零售额（本年累计）
func (s *MemoryStore) UpdateCompanyRetailCumulative(id string, newRetailCurrentCumulative float64) (*model.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}

	company.RetailCurrentCumulative = newRetailCurrentCumulative
	return cloneCompany(company), nil
}

// BatchUpdateCompanyRetailCumulative 批量更新企业零售额（本年累计）
func (s *MemoryStore) BatchUpdateCompanyRetailCumulative(updates map[string]float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, newValue := range updates {
		company, ok := s.companies[id]
		if !ok {
			continue
		}
		company.RetailCurrentCumulative = newValue
	}

	return nil
}

// BatchUpdateCompanySales 批量更新企业销售额（本期）
func (s *MemoryStore) BatchUpdateCompanySales(updates map[string]float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, newValue := range updates {
		company, ok := s.companies[id]
		if !ok {
			continue
		}
		delta := newValue - company.SalesCurrentMonth
		company.SalesCurrentMonth = newValue
		company.SalesCurrentCumulative += delta
	}

	return nil
}

// UpdateCompanySalesCumulative 更新企业销售额（本年累计）
func (s *MemoryStore) UpdateCompanySalesCumulative(id string, newSalesCurrentCumulative float64) (*model.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	company, ok := s.companies[id]
	if !ok {
		return nil, errors.New("company not found")
	}

	company.SalesCurrentCumulative = newSalesCurrentCumulative
	return cloneCompany(company), nil
}

// BatchUpdateCompanySalesCumulative 批量更新企业销售额（本年累计）
func (s *MemoryStore) BatchUpdateCompanySalesCumulative(updates map[string]float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, newValue := range updates {
		company, ok := s.companies[id]
		if !ok {
			continue
		}
		company.SalesCurrentCumulative = newValue
	}

	return nil
}

// ResetCompanies 重置企业数据
func (s *MemoryStore) ResetCompanies(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resetOne := func(c *model.Company) {
		if !c.OriginalInitialized {
			return
		}
		c.RowNo = c.OriginalRowNo
		c.Name = c.OriginalName

		c.RetailLastYearMonth = c.OriginalRetailLastYearMonth
		c.RetailCurrentMonth = c.OriginalRetailCurrentMonth
		c.RetailLastYearCumulative = c.OriginalRetailLastYearCumulative
		c.RetailCurrentCumulative = c.OriginalRetailCurrentCumulative

		c.SalesLastYearMonth = c.OriginalSalesLastYearMonth
		c.SalesCurrentMonth = c.OriginalSalesCurrentMonth
		c.SalesLastYearCumulative = c.OriginalSalesLastYearCumulative
		c.SalesCurrentCumulative = c.OriginalSalesCurrentCumulative

		c.RoomRevenueLastYearMonth = c.OriginalRoomRevenueLastYearMonth
		c.RoomRevenueCurrentMonth = c.OriginalRoomRevenueCurrentMonth
		c.RoomRevenueLastYearCumulative = c.OriginalRoomRevenueLastYearCumulative
		c.RoomRevenueCurrentCumulative = c.OriginalRoomRevenueCurrentCumulative

		c.FoodRevenueLastYearMonth = c.OriginalFoodRevenueLastYearMonth
		c.FoodRevenueCurrentMonth = c.OriginalFoodRevenueCurrentMonth
		c.FoodRevenueLastYearCumulative = c.OriginalFoodRevenueLastYearCumulative
		c.FoodRevenueCurrentCumulative = c.OriginalFoodRevenueCurrentCumulative

		c.GoodsSalesLastYearMonth = c.OriginalGoodsSalesLastYearMonth
		c.GoodsSalesCurrentMonth = c.OriginalGoodsSalesCurrentMonth
		c.GoodsSalesLastYearCumulative = c.OriginalGoodsSalesLastYearCumulative
		c.GoodsSalesCurrentCumulative = c.OriginalGoodsSalesCurrentCumulative
	}

	if len(ids) == 0 {
		// 重置全部
		for _, c := range s.companies {
			resetOne(c)
		}
	} else {
		// 重置指定企业
		for _, id := range ids {
			if c, ok := s.companies[id]; ok {
				resetOne(c)
			}
		}
	}
}

// GetConfig 获取配置
func (s *MemoryStore) GetConfig() *model.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.config)
}

// SetConfig 设置配置
func (s *MemoryStore) SetConfig(config *model.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cloneConfig(config)
}

// UpdateConfig 更新配置
func (s *MemoryStore) UpdateConfig(updates map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if v, ok := updates["currentMonth"].(int); ok {
		s.config.CurrentMonth = v
	}
	if v, ok := updates["lastYearLimitBelowCumulative"].(float64); ok {
		s.config.LastYearLimitBelowCumulative = v
	}
}

// Count 获取企业数量
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.companies)
}

// AddCompany 添加单个企业
func (s *MemoryStore) AddCompany(c *model.Company) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.OriginalRetailCurrentMonth = c.RetailCurrentMonth
	s.companies[c.ID] = c
}

// Clear 清空所有企业数据
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.companies = make(map[string]*model.Company)
}
