package calculator

import "sync"

// Dag 负责增量联动计算（最小可用版本：先覆盖“零售额本期汇总”）
type Dag struct {
	mu sync.RWMutex

	retailCurrentByCompany map[string]float64
	totalRetailCurrent     float64
}

// NewDag 创建 DAG
func NewDag() *Dag {
	return &Dag{
		retailCurrentByCompany: make(map[string]float64),
	}
}

// SetRetailCurrent 设置单个企业“本期零售额”
func (d *Dag) SetRetailCurrent(companyID string, value float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	old := d.retailCurrentByCompany[companyID]
	d.retailCurrentByCompany[companyID] = value
	d.totalRetailCurrent += value - old
}

// RecomputeFrom 从指定节点触发重算（占位：当前已在 Set 时增量维护）
func (d *Dag) RecomputeFrom(_ string) {
}

// GetTotalRetailCurrent 获取“本期零售额”全局汇总
func (d *Dag) GetTotalRetailCurrent() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.totalRetailCurrent
}
