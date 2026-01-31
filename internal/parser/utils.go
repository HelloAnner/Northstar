package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// ExtractYearMonth 从字符串中提取年月信息
// 支持格式: "2025年12月销售额" / "2025年12月" / "销售额;2025年12月"
func ExtractYearMonth(text string) (year, month int, found bool) {
	// 允许 "2024年;12月" / "2024年12月;xxx" 等分隔符形式
	re := regexp.MustCompile(`(\d{4})年[^0-9]{0,3}0?(\d{1,2})月`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		year, _ = strconv.Atoi(matches[1])
		month, _ = strconv.Atoi(matches[2])
		return year, month, true
	}
	return 0, 0, false
}

// ExtractMonthOnly 提取不带年份的月份
// 支持格式: "11月客房收入" / "12月销售额增速"
func ExtractMonthOnly(text string) (month int, found bool) {
	re := regexp.MustCompile(`(^|[^0-9])0?(\d{1,2})月`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		month, _ = strconv.Atoi(matches[2])
		return month, true
	}
	return 0, false
}

// ExtractYearMonthRange 提取年月范围
// 支持格式: "2025年1-12月" / "2025年1—12月"
func ExtractYearMonthRange(text string) (year, startMonth, endMonth int, found bool) {
	// 允许 "2024年;1-12月" / "2024年1-12月;xxx" 等分隔符形式
	re := regexp.MustCompile(`(\d{4})年[^0-9]{0,3}0?(\d{1,2})[-—]0?(\d{1,2})月`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 4 {
		year, _ = strconv.Atoi(matches[1])
		startMonth, _ = strconv.Atoi(matches[2])
		endMonth, _ = strconv.Atoi(matches[3])
		return year, startMonth, endMonth, true
	}
	return 0, 0, 0, false
}

// InferFieldTimeType 推断字段时间类型
func InferFieldTimeType(columnName string, currentYear, currentMonth int) FieldTimeType {
	// 检查是否包含累计关键词
	isCumulative := strings.Contains(columnName, "1-") ||
		strings.Contains(columnName, "1—") ||
		strings.Contains(columnName, "累计")

	// 尝试提取年月
	if isCumulative {
		// 累计字段
		year, _, endMonth, found := ExtractYearMonthRange(columnName)
		if found {
			if year == currentYear && endMonth == currentMonth {
				return CurrentCumulative
			}
			if year == currentYear && endMonth == currentMonth-1 {
				return PrevCumulative
			}
			if year == currentYear-1 && endMonth == currentMonth {
				return LastYearCumulative
			}
			if year == currentYear-1 && endMonth == currentMonth-1 {
				return LastYearPrevCumulative
			}
		}
		// 无法提取范围，通过年份判断
		year, _, found = ExtractYearMonth(columnName)
		if found {
			if year == currentYear {
				return CurrentCumulative
			} else if year == currentYear-1 {
				return LastYearCumulative
			}
		}
		// 通过关键词判断
		if strings.Contains(columnName, "上年") || strings.Contains(columnName, "去年") {
			return LastYearCumulative
		}
		return CurrentCumulative
	}

	// 单月字段
	year, month, found := ExtractYearMonth(columnName)
	if found {
		if year == currentYear {
			if month == currentMonth {
				return CurrentMonth
			} else if month == currentMonth-1 {
				return PrevMonth
			}
		} else if year == currentYear-1 {
			return LastYearMonth
		}
	}

	// 兼容：不带年份的 "11月xxx"
	month, found = ExtractMonthOnly(columnName)
	if found {
		if month == currentMonth {
			return CurrentMonth
		}
		if month == currentMonth-1 {
			return PrevMonth
		}
	}

	// 通过关键词判断
	if strings.Contains(columnName, "上年") || strings.Contains(columnName, "去年") {
		return LastYearMonth
	}
	if strings.Contains(columnName, "上月") {
		return PrevMonth
	}

	return CurrentMonth
}

// FindCurrentYearMonth 从列名列表中找出当前数据的年月
// 返回最大的年月组合
func FindCurrentYearMonth(columnNames []string) (year, month int) {
	maxYear := 0
	maxMonth := 0

	for _, col := range columnNames {
		y, m, found := ExtractYearMonth(col)
		if found {
			if y > maxYear || (y == maxYear && m > maxMonth) {
				maxYear = y
				maxMonth = m
			}
		}
	}

	return maxYear, maxMonth
}

// NormalizeColumnName 规范化列名，去除空格和特殊字符
func NormalizeColumnName(name string) string {
	// 去除首尾空格
	name = strings.TrimSpace(name)
	// 去除换行符和制表符
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\t", "")
	// 压缩多个空格为一个
	re := regexp.MustCompile(`\s+`)
	name = re.ReplaceAllString(name, "")
	return name
}

// ContainsAny 检查字符串是否包含任意一个关键词
func ContainsAny(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// MatchPattern 使用正则匹配
func MatchPattern(text, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(text)
}
