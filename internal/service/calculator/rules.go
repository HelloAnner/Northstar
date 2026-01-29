package calculator

import "northstar/internal/model"

// ValidateCompany 校验企业数据规则（用于联动调整与导出前校验）
func ValidateCompany(c *model.Company) []string {
	if c == nil {
		return []string{}
	}

	errs := make([]string, 0, 4)

	if c.RetailCurrentMonth < 0 {
		errs = append(errs, "零售额不能为负数")
	}
	if c.SalesCurrentMonth < 0 {
		errs = append(errs, "销售额不能为负数")
	}
	if c.SalesCurrentMonth > 0 && c.RetailCurrentMonth > c.SalesCurrentMonth {
		errs = append(errs, "零售额不能超过销售额")
	}

	return errs
}
