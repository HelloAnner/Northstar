package calculator

import (
	"errors"
	"sort"

	"northstar/internal/model"
	"northstar/internal/service/store"
)

// Optimizer 智能调整优化器
type Optimizer struct {
	store  *store.MemoryStore
	engine *Engine
}

// NewOptimizer 创建优化器
func NewOptimizer(store *store.MemoryStore, engine *Engine) *Optimizer {
	return &Optimizer{
		store:  store,
		engine: engine,
	}
}

// Optimize 执行智能调整
func (o *Optimizer) Optimize(targetRate float64, constraints *model.OptimizeConstraints) (*model.OptimizeResult, error) {
	if constraints == nil {
		constraints = model.DefaultOptimizeConstraints()
	}
	constraints.TargetGrowthRate = targetRate

	companies := o.store.GetAllCompanies()
	_ = o.store.GetConfig() // 预留配置使用

	// 当前累计增速
	currentSums := o.engine.GetSums()
	currentRate := calcRate(currentSums.AllRetailCurrentCumulative, currentSums.AllRetailLastYearCumulative)

	// 目标增速已达成
	if currentRate >= targetRate-0.0001 && currentRate <= targetRate+0.0001 {
		return &model.OptimizeResult{
			Success:       true,
			AchievedValue: currentRate,
			Adjustments:   []model.CompanyAdjustment{},
			Summary:       model.OptimizeSummary{},
			Indicators:    o.engine.Calculate(),
		}, nil
	}

	// 计算需要的总增量
	// target = (current_cumulative + delta - last_year_cumulative) / last_year_cumulative
	// delta = target * last_year_cumulative + last_year_cumulative - current_cumulative
	requiredDelta := targetRate*currentSums.AllRetailLastYearCumulative + currentSums.AllRetailLastYearCumulative - currentSums.AllRetailCurrentCumulative

	if requiredDelta < 0 {
		return nil, errors.New("目标增速低于当前值，不支持向下调整")
	}

	// 按优先级排序企业
	sortedCompanies := o.sortByPriority(companies, constraints.PriorityIndustries)

	// 贪心分配增量
	adjustments := []model.CompanyAdjustment{}
	updates := make(map[string]float64)
	remainingDelta := requiredDelta

	for _, c := range sortedCompanies {
		if remainingDelta <= 0 {
			break
		}

		// 计算该企业的最大可调整量
		maxRetail := c.SalesCurrentMonth // 零售额不能超过销售额
		if maxRetail <= 0 {
			maxRetail = c.RetailCurrentMonth * (1 + constraints.MaxIndividualRate)
		}

		// 约束: 增速不超过最大值
		maxByRate := c.RetailLastYearMonth * (1 + constraints.MaxIndividualRate)
		if maxByRate > 0 && maxByRate < maxRetail {
			maxRetail = maxByRate
		}

		// 可调整空间
		adjustable := maxRetail - c.RetailCurrentMonth
		if adjustable <= 0 {
			continue
		}

		// 实际调整量
		actualDelta := adjustable
		if actualDelta > remainingDelta {
			actualDelta = remainingDelta
		}

		newValue := c.RetailCurrentMonth + actualDelta
		updates[c.ID] = newValue

		adjustments = append(adjustments, model.CompanyAdjustment{
			CompanyID:     c.ID,
			CompanyName:   c.Name,
			OriginalValue: c.RetailCurrentMonth,
			AdjustedValue: newValue,
			ChangePercent: actualDelta / c.RetailCurrentMonth,
		})

		remainingDelta -= actualDelta
	}

	// 检查是否达成目标
	if remainingDelta > 0.01 {
		return &model.OptimizeResult{
			Success:       false,
			AchievedValue: targetRate - remainingDelta/currentSums.AllRetailLastYearCumulative,
			Adjustments:   adjustments,
			Summary: model.OptimizeSummary{
				AdjustedCount:   len(adjustments),
				TotalAdjustment: requiredDelta - remainingDelta,
			},
			Indicators: o.engine.Calculate(),
		}, errors.New("在当前约束条件下无法完全达成目标增速")
	}

	// 应用调整
	o.store.BatchUpdateCompanyRetail(updates)

	// 计算汇总
	totalAdjustment := 0.0
	totalChangePercent := 0.0
	for _, adj := range adjustments {
		totalAdjustment += adj.AdjustedValue - adj.OriginalValue
		totalChangePercent += adj.ChangePercent
	}

	indicators := o.engine.Calculate()

	return &model.OptimizeResult{
		Success:       true,
		AchievedValue: indicators.LimitAboveCumulativeRate,
		Adjustments:   adjustments,
		Summary: model.OptimizeSummary{
			AdjustedCount:        len(adjustments),
			TotalAdjustment:      totalAdjustment,
			AverageChangePercent: totalChangePercent / float64(len(adjustments)),
		},
		Indicators: indicators,
	}, nil
}

// Preview 预览智能调整结果（不实际修改数据）
func (o *Optimizer) Preview(targetRate float64, constraints *model.OptimizeConstraints) (*model.OptimizeResult, error) {
	// 保存当前状态
	companies := o.store.GetAllCompanies()
	originalValues := make(map[string]float64)
	for _, c := range companies {
		originalValues[c.ID] = c.RetailCurrentMonth
	}

	// 执行优化
	result, err := o.Optimize(targetRate, constraints)

	// 恢复原始状态
	o.store.BatchUpdateCompanyRetail(originalValues)

	return result, err
}

// sortByPriority 按优先级排序企业
func (o *Optimizer) sortByPriority(companies []*model.Company, priorityIndustries []string) []*model.Company {
	// 创建优先级映射
	priorityMap := make(map[string]int)
	for i, ind := range priorityIndustries {
		priorityMap[ind] = len(priorityIndustries) - i
	}

	sorted := make([]*model.Company, len(companies))
	copy(sorted, companies)

	sort.Slice(sorted, func(i, j int) bool {
		pi := priorityMap[string(sorted[i].IndustryType)]
		pj := priorityMap[string(sorted[j].IndustryType)]
		if pi != pj {
			return pi > pj
		}
		// 同优先级按可调整空间排序
		spaceI := sorted[i].SalesCurrentMonth - sorted[i].RetailCurrentMonth
		spaceJ := sorted[j].SalesCurrentMonth - sorted[j].RetailCurrentMonth
		return spaceI > spaceJ
	})

	return sorted
}
