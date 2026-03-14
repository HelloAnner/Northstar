/**
 * 规则过滤逻辑
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

func filterByMode(companies []CompanyRow, mode string) []CompanyRow {
	if len(companies) == 0 {
		return companies
	}
	if mode == "" {
		return companies
	}

	filtered := make([]CompanyRow, 0, len(companies))
	for _, company := range companies {
		if keepCompany(company, mode) {
			filtered = append(filtered, company)
		}
	}
	if len(filtered) == 0 && mode != "positive_current" && mode != "negative_current" &&
		mode != "large_scale_only" && mode != "exclude_small_micro" {
		return companies
	}
	if !isKnownFilterMode(mode) {
		return companies
	}
	return filtered
}

func keepCompany(company CompanyRow, mode string) bool {
	switch mode {
	case "positive_current":
		return company.CurrentValue > 0
	case "negative_current":
		return company.CurrentValue < 0
	case "large_scale_only":
		return company.CompanyScale == 1
	case "exclude_small_micro":
		return !company.IsSmallMicro
	default:
		return true
	}
}

func isKnownFilterMode(mode string) bool {
	return mode == "positive_current" || mode == "negative_current" ||
		mode == "large_scale_only" || mode == "exclude_small_micro"
}
