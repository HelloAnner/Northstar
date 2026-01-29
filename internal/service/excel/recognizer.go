package excel

import (
	"regexp"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
)

type headerRequirement struct {
	Key   string
	Match func(header string) bool
}

type sheetRule struct {
	Type         model.SheetType
	Requirements []headerRequirement
	NameBoost    func(sheetName string) float64
}

// Recognizer 工作簿 sheet 识别器
type Recognizer struct {
	rules []*sheetRule

	yearMonthRe     *regexp.Regexp
	yearSemicolonRe *regexp.Regexp
	monthOnlySemiRe *regexp.Regexp
	whitespaceRe    *regexp.Regexp
}

// NewRecognizer 创建识别器
func NewRecognizer() *Recognizer {
	r := &Recognizer{
		yearMonthRe:     regexp.MustCompile(`(\d{4})年(\d{1,2})月`),
		yearSemicolonRe: regexp.MustCompile(`(\d{4})年\s*;\s*(\d{1,2})月`),
		monthOnlySemiRe: regexp.MustCompile(`;\s*(\d{1,2})月\s*;`),
		whitespaceRe:    regexp.MustCompile(`\s+`),
	}
	r.rules = defaultSheetRules()
	return r
}

// RecognizeWorkbook 识别工作簿内每个 sheet 的类型
func (r *Recognizer) RecognizeWorkbook(wb *excelize.File) map[string]model.SheetRecognition {
	results := make(map[string]model.SheetRecognition)
	if wb == nil {
		return results
	}

	for _, sheetName := range wb.GetSheetList() {
		headers := readHeaderRow(wb, sheetName)
		normHeaders := make([]string, 0, len(headers))
		for _, h := range headers {
			if v := r.normalizeHeader(h); v != "" {
				normHeaders = append(normHeaders, v)
			}
		}

		bestType := model.SheetTypeUnknown
		bestScore := 0.0
		bestMissing := []string{}

		for _, rule := range r.rules {
			score, missing := scoreRule(rule, sheetName, normHeaders)
			if score > bestScore {
				bestType = rule.Type
				bestScore = score
				bestMissing = missing
			}
		}

		if bestScore < 0.50 {
			bestType = model.SheetTypeUnknown
		}

		results[sheetName] = model.SheetRecognition{
			SheetName:     sheetName,
			Type:          bestType,
			Score:         bestScore,
			MissingFields: bestMissing,
		}
	}

	return results
}

// ExtractMonths 从工作簿的 sheet 名/列名中提取可选月份列表（1-12）
func (r *Recognizer) ExtractMonths(wb *excelize.File) []int {
	if wb == nil {
		return []int{}
	}

	months := make(map[int]struct{})

	addFromText := func(text string) {
		for _, m := range r.extractMonthsFromText(text) {
			months[m] = struct{}{}
		}
	}

	for _, sheetName := range wb.GetSheetList() {
		addFromText(sheetName)
		for _, h := range readHeaderRow(wb, sheetName) {
			addFromText(h)
		}
	}

	out := make([]int, 0, len(months))
	for m := range months {
		out = append(out, m)
	}
	sort.Ints(out)
	return out
}

func (r *Recognizer) extractMonthsFromText(text string) []int {
	text = strings.TrimSpace(text)
	if text == "" {
		return []int{}
	}

	months := make(map[int]struct{})

	for _, m := range r.yearSemicolonRe.FindAllStringSubmatch(text, -1) {
		if len(m) >= 3 {
			month := parseMonth(m[2])
			if month > 0 {
				months[month] = struct{}{}
			}
		}
	}
	for _, m := range r.yearMonthRe.FindAllStringSubmatch(text, -1) {
		if len(m) >= 3 {
			month := parseMonth(m[2])
			if month > 0 {
				months[month] = struct{}{}
			}
		}
	}
	for _, m := range r.monthOnlySemiRe.FindAllStringSubmatch(text, -1) {
		if len(m) >= 2 {
			month := parseMonth(m[1])
			if month > 0 {
				months[month] = struct{}{}
			}
		}
	}

	out := make([]int, 0, len(months))
	for m := range months {
		out = append(out, m)
	}
	sort.Ints(out)
	return out
}

func (r *Recognizer) normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	s = strings.ReplaceAll(s, "；", ";")
	s = strings.ReplaceAll(s, "（", "(")
	s = strings.ReplaceAll(s, "）", ")")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "｜", ";")
	s = strings.ReplaceAll(s, "/", ";")
	s = strings.ReplaceAll(s, "、", ";")
	s = r.whitespaceRe.ReplaceAllString(s, " ")

	s = strings.ReplaceAll(s, "行业代码（GB/T4754-2017）", "行业代码(GB/T4754-2017)")

	s = r.yearSemicolonRe.ReplaceAllString(s, "YEAR_MONTH")
	s = r.yearMonthRe.ReplaceAllString(s, "YEAR_MONTH")
	return s
}

func readHeaderRow(wb *excelize.File, sheetName string) []string {
	rows, err := wb.GetRows(sheetName)
	if err != nil {
		return []string{}
	}
	if len(rows) == 0 {
		return []string{}
	}
	return rows[0]
}

func scoreRule(rule *sheetRule, sheetName string, headers []string) (float64, []string) {
	headerSet := make(map[string]struct{}, len(headers))
	for _, h := range headers {
		headerSet[h] = struct{}{}
	}

	hit := 0
	missing := make([]string, 0, len(rule.Requirements))
	for _, req := range rule.Requirements {
		ok := false
		for h := range headerSet {
			if req.Match(h) {
				ok = true
				break
			}
		}
		if ok {
			hit++
		} else {
			missing = append(missing, req.Key)
		}
	}

	if len(rule.Requirements) == 0 {
		return 0, missing
	}

	score := float64(hit) / float64(len(rule.Requirements))
	if rule.NameBoost != nil {
		score += rule.NameBoost(sheetName)
	}
	if score > 1.0 {
		score = 1.0
	}
	return score, missing
}

func defaultSheetRules() []*sheetRule {
	reqExact := func(key string) headerRequirement {
		return headerRequirement{
			Key:   key,
			Match: func(h string) bool { return h == key },
		}
	}
	reqContains := func(key string, subs ...string) headerRequirement {
		return headerRequirement{
			Key: key,
			Match: func(h string) bool {
				for _, s := range subs {
					if !strings.Contains(h, s) {
						return false
					}
				}
				return true
			},
		}
	}
	boostByKeyword := func(pairs map[string]float64) func(string) float64 {
		return func(sheetName string) float64 {
			for kw, v := range pairs {
				if strings.Contains(sheetName, kw) {
					return v
				}
			}
			return 0
		}
	}

	mainBase := []headerRequirement{
		reqExact("统一社会信用代码"),
		reqExact("单位详细名称"),
		reqContains("行业代码(GB/T4754-2017)", "行业代码(GB/T4754-2017)"),
		reqContains("YYYY年MM月销售额", "YEAR_MONTH", "销售额"),
		reqContains("YYYY年MM月零售额", "YEAR_MONTH", "零售额"),
		reqContains("YYYY年1-12月销售额", "1-12月", "销售额"),
		reqContains("YYYY年1-12月零售额", "1-12月", "零售额"),
	}

	snapshotWR := []headerRequirement{
		reqExact("统一社会信用代码"),
		reqExact("单位详细名称"),
		reqContains("商品销售额;本年-本月", "商品销售额", "本年-本月"),
		reqContains("零售额;本年-本月", "零售额", "本年-本月"),
		reqContains("商品销售额;本年-1-本月", "商品销售额", "本年-1", "本月"),
		reqContains("零售额;本年-1-本月", "零售额", "本年-1", "本月"),
		reqContains("单位规模", "单位规模"),
	}

	snapshotAC := []headerRequirement{
		reqExact("统一社会信用代码"),
		reqExact("单位详细名称"),
		reqContains("营业额;本年-本月", "营业额", "本年-本月"),
		reqContains("营业额;本年-1-本月", "营业额", "本年-1", "本月"),
		reqContains("客房收入;本年-本月", "客房收入", "本年-本月"),
		reqContains("餐费收入;本年-本月", "餐费收入", "本年-本月"),
	}

	eatWearUse := []headerRequirement{
		reqExact("处理地编码"),
		reqExact("统一社会信用代码"),
		reqExact("单位详细名称"),
		reqContains("行业代码(GB/T4754-2017)", "行业代码(GB/T4754-2017)"),
		reqContains("商品销售额;本年-本月", "商品销售额", "本年-本月"),
		reqContains("零售额;本年-本月", "零售额", "本年-本月"),
	}

	microSmall := []headerRequirement{
		reqExact("处理地编码"),
		reqExact("统一社会信用代码"),
		reqExact("单位详细名称"),
		reqContains("计算用小微当月零售额", "小微", "当月", "零售额"),
	}

	eatWearUseExcluded := []headerRequirement{
		reqExact("处理地编码"),
		reqExact("统一社会信用代码"),
		reqExact("单位详细名称"),
		reqContains("吃穿用当月零售额", "吃穿用", "当月", "零售额"),
	}

	return []*sheetRule{
		{
			Type:         model.SheetTypeWholesaleMain,
			Requirements: mainBase,
			NameBoost:    boostByKeyword(map[string]float64{"批发": 0.2}),
		},
		{
			Type:         model.SheetTypeRetailMain,
			Requirements: mainBase,
			NameBoost:    boostByKeyword(map[string]float64{"零售": 0.2}),
		},
		{
			Type:         model.SheetTypeAccommodationMain,
			Requirements: mainBase,
			NameBoost:    boostByKeyword(map[string]float64{"住宿": 0.2}),
		},
		{
			Type:         model.SheetTypeCateringMain,
			Requirements: mainBase,
			NameBoost:    boostByKeyword(map[string]float64{"餐饮": 0.2}),
		},
		{
			Type:         model.SheetTypeWholesaleRetailSnapshot,
			Requirements: snapshotWR,
			NameBoost:    boostByKeyword(map[string]float64{"批零": 0.2}),
		},
		{
			Type:         model.SheetTypeAccommodationCateringSnapshot,
			Requirements: snapshotAC,
			NameBoost:    boostByKeyword(map[string]float64{"住餐": 0.2}),
		},
		{
			Type:         model.SheetTypeEatWearUse,
			Requirements: eatWearUse,
			NameBoost:    boostByKeyword(map[string]float64{"吃穿用": 0.2}),
		},
		{
			Type:         model.SheetTypeMicroSmall,
			Requirements: microSmall,
			NameBoost:    boostByKeyword(map[string]float64{"小微": 0.2}),
		},
		{
			Type:         model.SheetTypeEatWearUseExcluded,
			Requirements: eatWearUseExcluded,
			NameBoost:    boostByKeyword(map[string]float64{"剔除": 0.2}),
		},
		{
			Type:         model.SheetTypeFixedSocialRetail,
			Requirements: []headerRequirement{},
			NameBoost:    boostByKeyword(map[string]float64{"社零额（定）": 1.0, "社零额(定)": 1.0}),
		},
		{
			Type:         model.SheetTypeFixedSummary,
			Requirements: []headerRequirement{},
			NameBoost:    boostByKeyword(map[string]float64{"汇总表（定）": 1.0, "汇总表(定)": 1.0}),
		},
	}
}

func parseMonth(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 || n > 12 {
		return 0
	}
	return n
}
