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
		result = append(result, c)
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
	return company, nil
}

// SetCompanies 设置企业列表
func (s *MemoryStore) SetCompanies(companies []*model.Company) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.companies = make(map[string]*model.Company)
	for _, c := range companies {
		// 保存原始值用于重置
		c.OriginalRetailCurrentMonth = c.RetailCurrentMonth
		s.companies[c.ID] = c
	}
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

	return company, nil
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

// ResetCompanies 重置企业数据
func (s *MemoryStore) ResetCompanies(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(ids) == 0 {
		// 重置全部
		for _, c := range s.companies {
			delta := c.OriginalRetailCurrentMonth - c.RetailCurrentMonth
			c.RetailCurrentMonth = c.OriginalRetailCurrentMonth
			c.RetailCurrentCumulative += delta
		}
	} else {
		// 重置指定企业
		for _, id := range ids {
			if c, ok := s.companies[id]; ok {
				delta := c.OriginalRetailCurrentMonth - c.RetailCurrentMonth
				c.RetailCurrentMonth = c.OriginalRetailCurrentMonth
				c.RetailCurrentCumulative += delta
			}
		}
	}
}

// GetConfig 获取配置
func (s *MemoryStore) GetConfig() *model.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SetConfig 设置配置
func (s *MemoryStore) SetConfig(config *model.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
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
