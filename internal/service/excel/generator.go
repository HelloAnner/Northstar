package excel

import (
	"math"
	"math/rand"
	"time"

	"northstar/internal/model"
)

// Generator 历史数据生成器
type Generator struct {
	rng *rand.Rand
}

// NewGenerator 创建生成器
func NewGenerator() *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateHistory 为新入库企业生成历史数据
func (g *Generator) GenerateHistory(company *model.Company, rule *model.GenerationRule) {
	if company.RetailLastYearMonth > 0 && company.RetailLastYearCumulative > 0 {
		// 已有历史数据，不生成
		return
	}

	if rule == nil {
		rule = g.defaultRule(company.IndustryType)
	}

	// 生成上年累计销售额目标值
	targetCumulative := g.randomInRange(rule.MinThreshold, rule.MaxThreshold)

	// 按月份拆分，引入季节性波动
	monthlyData := g.generateMonthlyData(targetCumulative, 12, rule.MonthlyVariance)

	// 设置上年同期（假设当前月份为6月）
	currentMonth := 6 // 可以从config获取
	company.RetailLastYearMonth = monthlyData[currentMonth-1]

	// 设置上年累计
	cumulative := 0.0
	for i := 0; i < currentMonth; i++ {
		cumulative += monthlyData[i]
	}
	company.RetailLastYearCumulative = cumulative

	// 同步设置销售额
	company.SalesLastYearMonth = company.RetailLastYearMonth * 1.1
	company.SalesLastYearCumulative = company.RetailLastYearCumulative * 1.1
}

// defaultRule 默认生成规则
func (g *Generator) defaultRule(industryType model.IndustryType) *model.GenerationRule {
	switch industryType {
	case model.IndustryWholesale:
		return &model.GenerationRule{
			MinThreshold:    20000000,
			MaxThreshold:    30000000,
			MonthlyVariance: 0.15,
		}
	case model.IndustryRetail:
		return &model.GenerationRule{
			MinThreshold:    5000000,
			MaxThreshold:    6000000,
			MonthlyVariance: 0.20,
		}
	case model.IndustryAccommodation:
		return &model.GenerationRule{
			MinThreshold:    2000000,
			MaxThreshold:    3000000,
			MonthlyVariance: 0.25,
		}
	case model.IndustryCatering:
		return &model.GenerationRule{
			MinThreshold:    2000000,
			MaxThreshold:    3000000,
			MonthlyVariance: 0.20,
		}
	default:
		return &model.GenerationRule{
			MinThreshold:    5000000,
			MaxThreshold:    6000000,
			MonthlyVariance: 0.20,
		}
	}
}

// randomInRange 在范围内生成随机数
func (g *Generator) randomInRange(min, max float64) float64 {
	return min + g.rng.Float64()*(max-min)
}

// generateMonthlyData 生成月度数据（带季节性波动）
func (g *Generator) generateMonthlyData(total float64, months int, variance float64) []float64 {
	data := make([]float64, months)
	baseValue := total / float64(months)

	sum := 0.0
	for i := 0; i < months; i++ {
		// 季节性波动：使用正弦函数模拟
		seasonalFactor := 1.0 + 0.15*math.Sin(float64(i)*math.Pi/6.0)
		// 随机噪声
		noiseFactor := 1.0 + (g.rng.Float64()-0.5)*variance*2
		data[i] = baseValue * seasonalFactor * noiseFactor
		sum += data[i]
	}

	// 调整使总和等于目标值
	ratio := total / sum
	for i := range data {
		data[i] *= ratio
	}

	return data
}

// BatchGenerateHistory 批量生成历史数据
func (g *Generator) BatchGenerateHistory(companies []*model.Company, rules map[model.IndustryType]*model.GenerationRule) int {
	generated := 0
	for _, c := range companies {
		if c.RetailLastYearMonth == 0 || c.RetailLastYearCumulative == 0 {
			rule := rules[c.IndustryType]
			g.GenerateHistory(c, rule)
			generated++
		}
	}
	return generated
}
