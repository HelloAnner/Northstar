/**
 * 导出模板输入单元格逻辑清单
 *
 * @author Anner
 * Created on 2026/2/5
 */

package exporter

import (
	"fmt"
	"math"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
	"northstar/internal/store"
)

type writeMode int

const (
	writeAlways writeMode = iota
	writeIfNoFormula
)

type cellValue struct {
	value interface{}
	write bool
}

func valueOf(v interface{}) cellValue {
	return cellValue{value: v, write: true}
}

func skipValue() cellValue {
	return cellValue{write: false}
}

type cellLogic[T any] struct {
	sheet string
	cell  string
	key   string
	desc  string
	mode  writeMode
	value func(*T) (cellValue, error)
}

type rowCellLogic[T any] struct {
	col   string
	key   string
	desc  string
	mode  writeMode
	value func(*T) (cellValue, error)
}

func applyCellLogics[T any](f *excelize.File, ctx *T, logics []cellLogic[T]) error {
	for _, it := range logics {
		cv, err := it.value(ctx)
		if err != nil {
			return err
		}
		if !cv.write {
			continue
		}
		if it.mode == writeIfNoFormula {
			if err := setCellValueIfNoFormula(f, it.sheet, it.cell, cv.value); err != nil {
				return err
			}
			continue
		}
		if err := setCellValue(f, it.sheet, it.cell, cv.value); err != nil {
			return err
		}
	}
	return nil
}

func applyRowLogics[T any](f *excelize.File, sheet string, row int, r *T, logics []rowCellLogic[T]) error {
	for _, it := range logics {
		cv, err := it.value(r)
		if err != nil {
			return err
		}
		if !cv.write {
			continue
		}
		cell := fmt.Sprintf("%s%d", it.col, row)
		if it.mode == writeIfNoFormula {
			if err := setCellValueIfNoFormula(f, sheet, cell, cv.value); err != nil {
				return err
			}
			continue
		}
		if err := setCellValue(f, sheet, cell, cv.value); err != nil {
			return err
		}
	}
	return nil
}

var wrRowLogics = []rowCellLogic[model.WholesaleRetail]{
	{col: "A", key: "{{wr.credit_code}}", desc: "统一社会信用代码", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.CreditCode)), nil
	}},
	{col: "B", key: "{{wr.name}}", desc: "单位名称", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.Name)), nil
	}},
	{col: "D", key: "{{wr.sales_current_month}}", desc: "本月销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesCurrentMonth)), nil
	}},
	{col: "E", key: "{{wr.sales_last_year_month}}", desc: "上年同期销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesLastYearMonth)), nil
	}},
	{col: "F", key: "{{wr.sales_month_rate}}", desc: "本月销售额增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.SalesCurrentMonth, r.SalesLastYearMonth)), nil
	}},
	{col: "G", key: "{{wr.sales_current_cumulative}}", desc: "本年累计销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesCurrentCumulative)), nil
	}},
	{col: "H", key: "{{wr.sales_last_year_cumulative}}", desc: "上年同期累计销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesLastYearCumulative)), nil
	}},
	{col: "I", key: "{{wr.sales_cumulative_rate}}", desc: "累计销售额增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.SalesCurrentCumulative, r.SalesLastYearCumulative)), nil
	}},
	{col: "J", key: "{{wr.retail_current_month}}", desc: "本月零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailCurrentMonth)), nil
	}},
	{col: "K", key: "{{wr.retail_last_year_month}}", desc: "上年同期零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailLastYearMonth)), nil
	}},
	{col: "L", key: "{{wr.零售业销售额增速_当月}}", desc: "本月零售额增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.RetailCurrentMonth, r.RetailLastYearMonth)), nil
	}},
	{col: "M", key: "{{wr.retail_current_cumulative}}", desc: "本年累计零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailCurrentCumulative)), nil
	}},
	{col: "N", key: "{{wr.retail_last_year_cumulative}}", desc: "上年同期累计零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailLastYearCumulative)), nil
	}},
	{col: "O", key: "{{wr.零售业销售额增速_累计}}", desc: "累计零售额增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.RetailCurrentCumulative, r.RetailLastYearCumulative)), nil
	}},
	{col: "P", key: "{{wr.first_report_ip}}", desc: "首次上报IP", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(r.FirstReportIP), nil
	}},
	{col: "Q", key: "{{wr.fill_ip}}", desc: "填报IP", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(r.FillIP), nil
	}},
}

var acRowLogics = []rowCellLogic[model.AccommodationCatering]{
	{col: "A", key: "{{ac.credit_code}}", desc: "统一社会信用代码", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.CreditCode)), nil
	}},
	{col: "B", key: "{{ac.name}}", desc: "单位名称", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.Name)), nil
	}},
	{col: "D", key: "{{ac.revenue_current_month}}", desc: "本月营业额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RevenueCurrentMonth)), nil
	}},
	{col: "E", key: "{{ac.revenue_last_year_month}}", desc: "上年同期营业额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RevenueLastYearMonth)), nil
	}},
	{col: "F", key: "{{ac.revenue_month_rate}}", desc: "本月营业额增速", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(ratePercent(r.RevenueCurrentMonth, r.RevenueLastYearMonth)), nil
	}},
	{col: "G", key: "{{ac.revenue_current_cumulative}}", desc: "本年累计营业额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RevenueCurrentCumulative)), nil
	}},
	{col: "H", key: "{{ac.revenue_last_year_cumulative}}", desc: "上年同期累计营业额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RevenueLastYearCumulative)), nil
	}},
	{col: "I", key: "{{ac.revenue_cumulative_rate}}", desc: "累计营业额增速", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(ratePercent(r.RevenueCurrentCumulative, r.RevenueLastYearCumulative)), nil
	}},
	{col: "J", key: "{{ac.room_current_month}}", desc: "本月客房收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RoomCurrentMonth)), nil
	}},
	{col: "K", key: "{{ac.room_last_year_month}}", desc: "上年同期客房收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RoomLastYearMonth)), nil
	}},
	{col: "L", key: "{{ac.room_current_cumulative}}", desc: "本年累计客房收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RoomCurrentCumulative)), nil
	}},
	{col: "M", key: "{{ac.room_last_year_cumulative}}", desc: "上年同期累计客房收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.RoomLastYearCumulative)), nil
	}},
	{col: "N", key: "{{ac.food_current_month}}", desc: "本月餐费收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.FoodCurrentMonth)), nil
	}},
	{col: "O", key: "{{ac.food_last_year_month}}", desc: "上年同期餐费收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.FoodLastYearMonth)), nil
	}},
	{col: "P", key: "{{ac.food_current_cumulative}}", desc: "本年累计餐费收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.FoodCurrentCumulative)), nil
	}},
	{col: "Q", key: "{{ac.food_last_year_cumulative}}", desc: "上年同期累计餐费收入", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.FoodLastYearCumulative)), nil
	}},
	{col: "R", key: "{{ac.goods_current_month}}", desc: "本月商品销售", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.GoodsCurrentMonth)), nil
	}},
	{col: "S", key: "{{ac.goods_last_year_month}}", desc: "上年同期商品销售", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.GoodsLastYearMonth)), nil
	}},
	{col: "T", key: "{{ac.goods_current_cumulative}}", desc: "本年累计商品销售", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.GoodsCurrentCumulative)), nil
	}},
	{col: "U", key: "{{ac.goods_last_year_cumulative}}", desc: "上年同期累计商品销售", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		return valueOf(math.Round(r.GoodsLastYearCumulative)), nil
	}},
	{col: "V", key: "{{ac.retail_current_month}}", desc: "本月零售额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		retailCur := math.Round(r.FoodCurrentMonth + r.GoodsCurrentMonth)
		return valueOf(retailCur), nil
	}},
	{col: "W", key: "{{ac.retail_last_year_month}}", desc: "上年同期零售额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		retailLast := math.Round(r.FoodLastYearMonth + r.GoodsLastYearMonth)
		return valueOf(retailLast), nil
	}},
	{col: "X", key: "{{ac.retail_current_cumulative}}", desc: "本年累计零售额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		retailCurCum := math.Round(r.FoodCurrentCumulative + r.GoodsCurrentCumulative)
		return valueOf(retailCurCum), nil
	}},
	{col: "Y", key: "{{ac.retail_last_year_cumulative}}", desc: "上年同期累计零售额", mode: writeAlways, value: func(r *model.AccommodationCatering) (cellValue, error) {
		retailLastCum := math.Round(r.FoodLastYearCumulative + r.GoodsLastYearCumulative)
		return valueOf(retailLastCum), nil
	}},
}

var eatWearUseRowLogics = []rowCellLogic[model.WholesaleRetail]{
	{col: "B", key: "{{ewu.credit_code}}", desc: "统一社会信用代码", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.CreditCode)), nil
	}},
	{col: "C", key: "{{ewu.name}}", desc: "单位名称", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.Name)), nil
	}},
	{col: "D", key: "{{ewu.industry_code}}", desc: "行业代码", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(normalizeCodeText(r.IndustryCode)), nil
	}},
	{col: "E", key: "{{ewu.sales_current_month}}", desc: "本月销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesCurrentMonth)), nil
	}},
	{col: "F", key: "{{ewu.sales_last_year_month}}", desc: "上年同期销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesLastYearMonth)), nil
	}},
	{col: "G", key: "{{ewu.sales_month_rate}}", desc: "销售额当月增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.SalesCurrentMonth, r.SalesLastYearMonth)), nil
	}},
	{col: "H", key: "{{ewu.sales_current_cumulative}}", desc: "本年累计销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesCurrentCumulative)), nil
	}},
	{col: "I", key: "{{ewu.sales_last_year_cumulative}}", desc: "上年同期累计销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.SalesLastYearCumulative)), nil
	}},
	{col: "J", key: "{{ewu.sales_cumulative_rate}}", desc: "销售额累计增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.SalesCurrentCumulative, r.SalesLastYearCumulative)), nil
	}},
	{col: "K", key: "{{ewu.retail_current_month}}", desc: "本月零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailCurrentMonth)), nil
	}},
	{col: "L", key: "{{ewu.retail_last_year_month}}", desc: "上年同期零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailLastYearMonth)), nil
	}},
	{col: "M", key: "{{ewu.零售业销售额增速_当月}}", desc: "零售额当月增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.RetailCurrentMonth, r.RetailLastYearMonth)), nil
	}},
	{col: "N", key: "{{ewu.retail_current_cumulative}}", desc: "本年累计零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailCurrentCumulative)), nil
	}},
	{col: "O", key: "{{ewu.retail_last_year_cumulative}}", desc: "上年同期累计零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailLastYearCumulative)), nil
	}},
	{col: "P", key: "{{ewu.零售业销售额增速_累计}}", desc: "零售额累计增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.RetailCurrentCumulative, r.RetailLastYearCumulative)), nil
	}},
	{col: "Q", key: "{{ewu.eat_wear_use_current}}", desc: "吃穿用本月零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		retailCur := math.Round(r.RetailCurrentMonth)
		if r.IsEatWearUse == 1 {
			return valueOf(retailCur), nil
		}
		return valueOf(0.0), nil
	}},
	{col: "R", key: "{{ewu.eat_wear_use_last}}", desc: "吃穿用上年同期零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		retailLast := math.Round(r.RetailLastYearMonth)
		if r.IsEatWearUse == 1 {
			return valueOf(retailLast), nil
		}
		return valueOf(0.0), nil
	}},
	{col: "S", key: "{{ewu.eat_wear_use_rate}}", desc: "吃穿用增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		retailCur := 0.0
		retailLast := 0.0
		if r.IsEatWearUse == 1 {
			retailCur = math.Round(r.RetailCurrentMonth)
			retailLast = math.Round(r.RetailLastYearMonth)
		}
		return valueOf(ratePercent(retailCur, retailLast)), nil
	}},
	{col: "T", key: "{{ewu.company_scale}}", desc: "企业规模", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(r.CompanyScale), nil
	}},
	{col: "U", key: "{{ewu.micro_current}}", desc: "小微本月零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		retailCur := math.Round(r.RetailCurrentMonth)
		if r.IsSmallMicro == 1 {
			return valueOf(retailCur), nil
		}
		return valueOf(0.0), nil
	}},
	{col: "V", key: "{{ewu.micro_last}}", desc: "小微上年同期零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		retailLast := math.Round(r.RetailLastYearMonth)
		if r.IsSmallMicro == 1 {
			return valueOf(retailLast), nil
		}
		return valueOf(0.0), nil
	}},
	{col: "W", key: "{{ewu.network_sales}}", desc: "网络销售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.NetworkSales)), nil
	}},
	{col: "X", key: "{{ewu.placeholder}}", desc: "固定值", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(0), nil
	}},
	{col: "Y", key: "{{ewu.opening_year}}", desc: "开业年份", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		if r.OpeningYear == nil {
			return skipValue(), nil
		}
		return valueOf(*r.OpeningYear), nil
	}},
	{col: "Z", key: "{{ewu.opening_month}}", desc: "开业月份", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		if r.OpeningMonth == nil {
			return skipValue(), nil
		}
		return valueOf(*r.OpeningMonth), nil
	}},
}

var microSmallRowLogics = []rowCellLogic[model.WholesaleRetail]{
	{col: "B", key: "{{micro.credit_code}}", desc: "统一社会信用代码", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.CreditCode)), nil
	}},
	{col: "C", key: "{{micro.name}}", desc: "单位名称", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(strings.TrimSpace(r.Name)), nil
	}},
	{col: "D", key: "{{micro.retail_current_month}}", desc: "本月零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailCurrentMonth)), nil
	}},
	{col: "E", key: "{{micro.retail_last_year_month}}", desc: "上年同期零售额", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(math.Round(r.RetailLastYearMonth)), nil
	}},
	{col: "F", key: "{{micro.零售业销售额增速_当月}}", desc: "当月增速", mode: writeAlways, value: func(r *model.WholesaleRetail) (cellValue, error) {
		return valueOf(ratePercent(r.RetailCurrentMonth, r.RetailLastYearMonth)), nil
	}},
}

type socialContext struct {
	store          *store.Store
	year           int
	month          int
	indicators     indicatorIndex
	prevIndicators indicatorIndex
	retailCache    map[string]float64
}

func newSocialContext(st *store.Store, year, month int, indicators indicatorIndex) *socialContext {
	prevYear, prevMonth := prevYearMonth(year, month)
	prevIndicators, err := calculateIndicatorIndex(st, prevYear, prevMonth)
	if err != nil {
		prevIndicators = indicatorIndex{}
	}
	return &socialContext{
		store:          st,
		year:           year,
		month:          month,
		indicators:     indicators,
		prevIndicators: prevIndicators,
		retailCache:    map[string]float64{},
	}
}

func (c *socialContext) configFloat(key string) float64 {
	v, err := c.store.GetConfigFloat(key)
	if err != nil {
		return 0
	}
	return v
}

func (c *socialContext) platformRetailCur(year, month int) (float64, error) {
	key := fmt.Sprintf("%d-%02d", year, month)
	if v, ok := c.retailCache[key]; ok {
		return v, nil
	}
	wrRecords, err := c.store.GetWRByYearMonth(store.WRQueryOptions{DataYear: &year, DataMonth: &month})
	if err != nil {
		return 0, err
	}
	acRecords, err := c.store.GetACByYearMonth(store.ACQueryOptions{DataYear: &year, DataMonth: &month})
	if err != nil {
		return 0, err
	}
	retail := 0.0
	for _, r := range wrRecords {
		retail += math.Round(r.RetailCurrentMonth)
	}
	for _, r := range acRecords {
		retail += math.Round(r.FoodCurrentMonth + r.GoodsCurrentMonth)
	}
	c.retailCache[key] = retail
	return retail, nil
}

func (c *socialContext) platformRetailCurOrZero(year, month int) float64 {
	v, err := c.platformRetailCur(year, month)
	if err != nil {
		return 0
	}
	return v
}

func socialSheetLogics() []cellLogic[socialContext] {
	return []cellLogic[socialContext]{
		{sheet: "社零额（定）", cell: "A1", key: "{{social.title}}", desc: "标题", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(fmt.Sprintf("社零额计算举例（%d年%d月数据）", ctx.year, ctx.month)), nil
		}},
		{sheet: "社零额（定）", cell: "J2", key: "{{social.month}}", desc: "月份", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.month), nil
		}},
		{sheet: "社零额（定）", cell: "B4", key: "{{social.micro_rate_month}}", desc: "小微增速（本月）", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.indicators["小微企业增速_当月"].Value), nil
		}},
		{sheet: "社零额（定）", cell: "C4", key: "{{social.eat_wear_use_rate_month}}", desc: "吃穿用增速（本月）", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.indicators["吃穿用增速_当月"].Value), nil
		}},
		{sheet: "社零额（定）", cell: "D4", key: "{{social.sample_rate_month}}", desc: "抽样增速（本月）", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("sample_rate_month")), nil
		}},
		{sheet: "社零额（定）", cell: "B6", key: "{{social.micro_rate_prev}}", desc: "小微增速（上月）", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.prevIndicators["小微企业增速_当月"].Value), nil
		}},
		{sheet: "社零额（定）", cell: "C6", key: "{{social.eat_wear_use_rate_prev}}", desc: "吃穿用增速（上月）", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.prevIndicators["吃穿用增速_当月"].Value), nil
		}},
		{sheet: "社零额（定）", cell: "D6", key: "{{social.sample_rate_prev}}", desc: "抽样增速（上月）", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("sample_rate_prev")), nil
		}},
		{sheet: "社零额（定）", cell: "B12", key: "{{social.weight_small_micro}}", desc: "权重-小微", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("weight_small_micro")), nil
		}},
		{sheet: "社零额（定）", cell: "C12", key: "{{social.weight_eat_wear_use}}", desc: "权重-吃穿用", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("weight_eat_wear_use")), nil
		}},
		{sheet: "社零额（定）", cell: "D12", key: "{{social.weight_sample}}", desc: "权重-抽样", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("weight_sample")), nil
		}},
		{sheet: "社零额（定）", cell: "I3", key: "{{social.province_limit_below_rate_change}}", desc: "省限下增速变动", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("province_limit_below_rate_change")), nil
		}},
		{sheet: "社零额（定）", cell: "E18", key: "{{social.history_e18}}", desc: "历史社零额E18", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("history_social_e18")), nil
		}},
		{sheet: "社零额（定）", cell: "E19", key: "{{social.history_e19}}", desc: "历史社零额E19", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("history_social_e19")), nil
		}},
		{sheet: "社零额（定）", cell: "E20", key: "{{social.history_e20}}", desc: "历史社零额E20", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("history_social_e20")), nil
		}},
		{sheet: "社零额（定）", cell: "E21", key: "{{social.history_e21}}", desc: "历史社零额E21", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("history_social_e21")), nil
		}},
		{sheet: "社零额（定）", cell: "E22", key: "{{social.history_e22}}", desc: "历史社零额E22", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("history_social_e22")), nil
		}},
		{sheet: "社零额（定）", cell: "E23", key: "{{social.history_e23}}", desc: "历史社零额E23", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.configFloat("history_social_e23")), nil
		}},
		{sheet: "社零额（定）", cell: "K4", key: "{{social.platform_retail_prev_month}}", desc: "上月平台上报零售额", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			prevYear, prevMonth := prevYearMonth(ctx.year, ctx.month)
			return valueOf(ctx.platformRetailCurOrZero(prevYear, prevMonth)), nil
		}},
		{sheet: "社零额（定）", cell: "K6", key: "{{social.platform_retail_prev_year_prev_month}}", desc: "上年同期上月平台零售额", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			_, prevMonth := prevYearMonth(ctx.year, ctx.month)
			return valueOf(ctx.platformRetailCurOrZero(ctx.year-1, prevMonth)), nil
		}},
		{sheet: "社零额（定）", cell: "K13", key: "{{social.platform_retail_last_year_month}}", desc: "上年同期平台零售额", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			return valueOf(ctx.platformRetailCurOrZero(ctx.year-1, ctx.month)), nil
		}},
		{sheet: "社零额（定）", cell: "K16", key: "{{social.platform_retail_current_month}}", desc: "本月平台上报零售额", mode: writeIfNoFormula, value: func(ctx *socialContext) (cellValue, error) {
			v, err := ctx.platformRetailCur(ctx.year, ctx.month)
			if err != nil {
				return skipValue(), err
			}
			return valueOf(v), nil
		}},
	}
}

type summaryContext struct {
	year                   int
	month                  int
	wh                     wrSums
	re                     wrSums
	acc                    wrSums
	cat                    wrSums
	indicators             indicatorIndex
	totalCompanies         int
	reportedCompanies      int
	negativeGrowthCount    int
	reportRate             float64
	reportText             string
	overallRetailCur       float64
	overallRetailLast      float64
	overallRetailCurCum    float64
	overallRetailLastCum   float64
	limitAboveMonthWan     float64
	limitAboveLastMonthWan float64
	limitAboveCumWan       float64
	limitAboveLastCumWan   float64
	title                  string
	summaryText            string
}

func newSummaryContext(
	year int,
	month int,
	wh wrSums,
	re wrSums,
	acc wrSums,
	cat wrSums,
	indicators indicatorIndex,
	wrRecords []*model.WholesaleRetail,
	acRecords []*model.AccommodationCatering,
) *summaryContext {
	totalCompanies := len(wrRecords) + len(acRecords)
	reportedCompanies := totalCompanies
	negativeGrowthCount := 0
	for _, r := range wrRecords {
		if r.SalesLastYearMonth > 0 && r.SalesCurrentMonth < r.SalesLastYearMonth {
			negativeGrowthCount++
		}
	}
	for _, r := range acRecords {
		if r.RevenueLastYearMonth > 0 && r.RevenueCurrentMonth < r.RevenueLastYearMonth {
			negativeGrowthCount++
		}
	}

	reportRate := 0.0
	if totalCompanies > 0 {
		reportRate = float64(reportedCompanies) / float64(totalCompanies) * 100.0
	}
	if math.IsNaN(reportRate) || math.IsInf(reportRate, 0) {
		reportRate = 0
	}
	statusText := "已全部上报"
	if reportRate != 100 {
		statusText = fmt.Sprintf("已上报%d家，上报进度%s%%", reportedCompanies, formatTrimFloat(reportRate, 1))
	}

	overallRetailCur := wh.retailCur + re.retailCur + acc.retailCur + cat.retailCur
	overallRetailLast := wh.retailLast + re.retailLast + acc.retailLast + cat.retailLast
	overallRetailCurCum := wh.retailCurCum + re.retailCurCum + acc.retailCurCum + cat.retailCurCum
	overallRetailLastCum := wh.retailLastCum + re.retailLastCum + acc.retailLastCum + cat.retailLastCum

	// 汇总表（定）口径为“万元”，行业表/总表口径为“千元”
	limitAboveMonthWan := math.Round(overallRetailCur / 10.0)
	limitAboveLastMonthWan := math.Round(overallRetailLast / 10.0)
	limitAboveCumWan := math.Round(overallRetailCurCum / 10.0)
	limitAboveLastCumWan := math.Round(overallRetailLastCum / 10.0)

	title := fmt.Sprintf("%d年1-%02d月限上单位上报情况表", year, month)
	monthText := excelMid(title, 8, 1)
	periodText := excelMid(title, 6, 4)

	summaryText := fmt.Sprintf(
		"社会消费品零售总额：全县%d家限上商贸单位%s。%s月，批发、零售、住宿、餐饮业销售额(营业额)同比分别增长%s%%、%s%%、%s%%、%s%%。当月上报零售额%s亿元，同比增长%s%%；累计上报零售额%s亿元，同比增长%s%%。%s，全社会消费品零售总额预计完成%s亿元，同比增长%s%%。",
		totalCompanies,
		statusText,
		monthText,
		formatTrimFloat(ratePercent(wh.salesCur, wh.salesLast), 1),
		formatTrimFloat(ratePercent(re.salesCur, re.salesLast), 1),
		formatTrimFloat(ratePercent(acc.salesCur, acc.salesLast), 1),
		formatTrimFloat(ratePercent(cat.salesCur, cat.salesLast), 1),
		formatTrimFloat(limitAboveMonthWan/10000.0, 2),
		formatTrimFloat(ratePercent(overallRetailCur, overallRetailLast), 1),
		formatTrimFloat(limitAboveCumWan/10000.0, 2),
		formatTrimFloat(ratePercent(overallRetailCurCum, overallRetailLastCum), 1),
		periodText,
		formatTrimFloat(roundHalfUp(indicators["社零总额_累计值"].Value/10000.0, 2), 2),
		formatTrimFloat(indicators["社零总额增速_累计"].Value, 1),
	)

	return &summaryContext{
		year:                   year,
		month:                  month,
		wh:                     wh,
		re:                     re,
		acc:                    acc,
		cat:                    cat,
		indicators:             indicators,
		totalCompanies:         totalCompanies,
		reportedCompanies:      reportedCompanies,
		negativeGrowthCount:    negativeGrowthCount,
		reportRate:             reportRate,
		reportText:             statusText,
		overallRetailCur:       overallRetailCur,
		overallRetailLast:      overallRetailLast,
		overallRetailCurCum:    overallRetailCurCum,
		overallRetailLastCum:   overallRetailLastCum,
		limitAboveMonthWan:     limitAboveMonthWan,
		limitAboveLastMonthWan: limitAboveLastMonthWan,
		limitAboveCumWan:       limitAboveCumWan,
		limitAboveLastCumWan:   limitAboveLastCumWan,
		title:                  title,
		summaryText:            summaryText,
	}
}

func excelMid(s string, start, length int) string {
	if start <= 0 || length <= 0 {
		return ""
	}
	runes := []rune(s)
	startIndex := start - 1
	if startIndex >= len(runes) {
		return ""
	}
	endIndex := startIndex + length
	if endIndex > len(runes) {
		endIndex = len(runes)
	}
	return string(runes[startIndex:endIndex])
}

func summarySheetLogics() []cellLogic[summaryContext] {
	return []cellLogic[summaryContext]{
		{sheet: "汇总表（定）", cell: "A1", key: "{{summary.title}}", desc: "标题", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.title), nil
		}},
		{sheet: "汇总表（定）", cell: "B4", key: "{{summary.total_companies}}", desc: "单位总数", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.totalCompanies), nil
		}},
		{sheet: "汇总表（定）", cell: "C4", key: "{{summary.reported_companies}}", desc: "已上报数", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.reportedCompanies), nil
		}},
		{sheet: "汇总表（定）", cell: "E4", key: "{{summary.negative_growth_count}}", desc: "负增长企业数", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.negativeGrowthCount), nil
		}},
		{sheet: "汇总表（定）", cell: "G4", key: "{{summary.limit_above_month_wan}}", desc: "限上零售额当月（万元）", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.limitAboveMonthWan), nil
		}},
		{sheet: "汇总表（定）", cell: "H4", key: "{{summary.limit_above_last_month_wan}}", desc: "限上零售额上年同期当月（万元）", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.limitAboveLastMonthWan), nil
		}},
		{sheet: "汇总表（定）", cell: "I4", key: "{{summary.limit_above_cum_wan}}", desc: "限上零售额累计（万元）", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.limitAboveCumWan), nil
		}},
		{sheet: "汇总表（定）", cell: "J4", key: "{{summary.limit_above_last_cum_wan}}", desc: "限上零售额上年同期累计（万元）", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.limitAboveLastCumWan), nil
		}},
		{sheet: "汇总表（定）", cell: "K4", key: "{{summary.批发业销售额增速_当月}}", desc: "批发当月增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.wh.salesCur, ctx.wh.salesLast)), nil
		}},
		{sheet: "汇总表（定）", cell: "L4", key: "{{summary.wholesale_cum_rate}}", desc: "批发累计增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.wh.salesCurCum, ctx.wh.salesLastCum)), nil
		}},
		{sheet: "汇总表（定）", cell: "M4", key: "{{summary.零售业销售额增速_当月}}", desc: "零售当月增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.re.salesCur, ctx.re.salesLast)), nil
		}},
		{sheet: "汇总表（定）", cell: "N4", key: "{{summary.retail_cum_rate}}", desc: "零售累计增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.re.salesCurCum, ctx.re.salesLastCum)), nil
		}},
		{sheet: "汇总表（定）", cell: "O4", key: "{{summary.住宿业营业额增速_当月}}", desc: "住宿当月增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.acc.salesCur, ctx.acc.salesLast)), nil
		}},
		{sheet: "汇总表（定）", cell: "P4", key: "{{summary.accommodation_cum_rate}}", desc: "住宿累计增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.acc.salesCurCum, ctx.acc.salesLastCum)), nil
		}},
		{sheet: "汇总表（定）", cell: "Q4", key: "{{summary.餐饮业营业额增速_当月}}", desc: "餐饮当月增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.cat.salesCur, ctx.cat.salesLast)), nil
		}},
		{sheet: "汇总表（定）", cell: "R4", key: "{{summary.catering_cum_rate}}", desc: "餐饮累计增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.cat.salesCurCum, ctx.cat.salesLastCum)), nil
		}},
		{sheet: "汇总表（定）", cell: "S4", key: "{{summary.overall_零售业销售额增速_当月}}", desc: "限上零售额当月增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.overallRetailCur, ctx.overallRetailLast)), nil
		}},
		{sheet: "汇总表（定）", cell: "T4", key: "{{summary.overall_retail_cum_rate}}", desc: "限上零售额累计增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ratePercent(ctx.overallRetailCurCum, ctx.overallRetailLastCum)), nil
		}},
		{sheet: "汇总表（定）", cell: "U4", key: "{{summary.eat_wear_use_rate}}", desc: "吃穿用增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.indicators["吃穿用增速_当月"].Value), nil
		}},
		{sheet: "汇总表（定）", cell: "V4", key: "{{summary.micro_small_rate}}", desc: "小微增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.indicators["小微企业增速_当月"].Value), nil
		}},
		{sheet: "汇总表（定）", cell: "W4", key: "{{summary.note_time}}", desc: "备注时间", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(fmt.Sprintf("%d-%02d", ctx.year, ctx.month)), nil
		}},
		{sheet: "汇总表（定）", cell: "N10", key: "{{summary.total_social_yi}}", desc: "社零额预计（亿元）", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(roundHalfUp(ctx.indicators["社零总额_累计值"].Value/10000.0, 2)), nil
		}},
		{sheet: "汇总表（定）", cell: "S10", key: "{{summary.total_social_rate}}", desc: "社零额增速", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.indicators["社零总额增速_累计"].Value), nil
		}},
		{sheet: "汇总表（定）", cell: "X3", key: "{{summary.text}}", desc: "汇总文案", mode: writeIfNoFormula, value: func(ctx *summaryContext) (cellValue, error) {
			return valueOf(ctx.summaryText), nil
		}},
	}
}
