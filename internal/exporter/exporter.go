package exporter

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
	"northstar/internal/store"
)

// Exporter 月报导出器（定稿）
//
// 强约束：必须以“定稿模板”作为输出模板，仅填充数据区域，保留模板的 sheet、合并单元格、列宽、字体、颜色、边框与公式位置。
type Exporter struct {
	store        *store.Store
	templatePath string
}

// NewExporter 创建导出器
func NewExporter(store *store.Store, templatePath string) *Exporter {
	return &Exporter{
		store:        store,
		templatePath: templatePath,
	}
}

// ExportOptions 导出选项
type ExportOptions struct {
	Year  int
	Month int
}

// Export 导出 Excel
func (e *Exporter) Export(opts ExportOptions) (*excelize.File, error) {
	f, err := e.openTemplateWorkbook()
	if err != nil {
		return nil, err
	}

	if err := e.fillTemplateWorkbook(f, opts); err != nil {
		_ = f.Close()
		return nil, err
	}

	f.SetActiveSheet(0)
	return f, nil
}

func (e *Exporter) openTemplateWorkbook() (*excelize.File, error) {
	// 优先使用外部模板（用于调试/版本切换）；未配置时使用内置模板（线上不依赖文件）。
	if p := strings.TrimSpace(e.templatePath); p != "" {
		f, err := excelize.OpenFile(p)
		if err != nil {
			return nil, fmt.Errorf("打开定稿模板失败: %w", err)
		}
		return f, nil
	}
	if v := strings.TrimSpace(os.Getenv("NORTHSTAR_EXCEL_TEMPLATE_PATH")); v != "" {
		f, err := excelize.OpenFile(v)
		if err != nil {
			return nil, fmt.Errorf("打开定稿模板失败: %w", err)
		}
		return f, nil
	}
	if v := strings.TrimSpace(os.Getenv("NS_MONTH_REPORT_TEMPLATE_XLSX")); v != "" {
		f, err := excelize.OpenFile(v)
		if err != nil {
			return nil, fmt.Errorf("打开定稿模板失败: %w", err)
		}
		return f, nil
	}

	f, err := openEmbeddedMonthReportTemplate()
	if err != nil {
		return nil, fmt.Errorf("打开内置定稿模板失败: %w", err)
	}
	return f, nil
}

func (e *Exporter) fillTemplateWorkbook(f *excelize.File, opts ExportOptions) error {
	wrRecords, err := e.store.GetWRByYearMonth(store.WRQueryOptions{
		DataYear:  &opts.Year,
		DataMonth: &opts.Month,
	})
	if err != nil {
		return fmt.Errorf("读取批零数据失败: %w", err)
	}

	acRecords, err := e.store.GetACByYearMonth(store.ACQueryOptions{
		DataYear:  &opts.Year,
		DataMonth: &opts.Month,
	})
	if err != nil {
		return fmt.Errorf("读取住餐数据失败: %w", err)
	}

	if err := e.fillWholesaleRetailSheets(f, wrRecords); err != nil {
		return err
	}
	if err := e.fillAccommodationCateringSheets(f, acRecords); err != nil {
		return err
	}

	if err := e.rewriteFixedTotals(f); err != nil {
		return err
	}

	return nil
}

func (e *Exporter) fillWholesaleRetailSheets(f *excelize.File, records []*model.WholesaleRetail) error {
	var wholesale []*model.WholesaleRetail
	var retail []*model.WholesaleRetail
	for _, r := range records {
		switch strings.TrimSpace(r.IndustryType) {
		case "wholesale":
			wholesale = append(wholesale, r)
		case "retail":
			retail = append(retail, r)
		default:
			continue
		}
	}

	if err := fillWRSheetByIndustryCodeOrder(f, "批发", wholesale); err != nil {
		return fmt.Errorf("写入 批发 失败: %w", err)
	}
	if err := fillWRSheetByIndustryCodeOrder(f, "零售", retail); err != nil {
		return fmt.Errorf("写入 零售 失败: %w", err)
	}
	if err := fillWRSheetByIndustryCodeOrder(f, "批零总表", records); err != nil {
		return fmt.Errorf("写入 批零总表 失败: %w", err)
	}

	return nil
}

func (e *Exporter) fillAccommodationCateringSheets(f *excelize.File, records []*model.AccommodationCatering) error {
	var accommodation []*model.AccommodationCatering
	var catering []*model.AccommodationCatering
	for _, r := range records {
		switch strings.TrimSpace(r.IndustryType) {
		case "accommodation":
			accommodation = append(accommodation, r)
		case "catering":
			catering = append(catering, r)
		default:
			continue
		}
	}

	if err := fillACIndustrySheetByIndustryCodeOrder(f, "住宿", accommodation); err != nil {
		return fmt.Errorf("写入 住宿 失败: %w", err)
	}
	if err := fillACIndustrySheetByIndustryCodeOrder(f, "餐饮", catering); err != nil {
		return fmt.Errorf("写入 餐饮 失败: %w", err)
	}
	if err := fillACTotalSheetByIndustryCodeOrder(f, "住餐总表", records); err != nil {
		return fmt.Errorf("写入 住餐总表 失败: %w", err)
	}

	return nil
}

type wrSums struct {
	maxRow        int
	salesCur      float64
	salesLast     float64
	salesCurCum   float64
	salesLastCum  float64
	retailCur     float64
	retailLast    float64
	retailCurCum  float64
	retailLastCum float64
}

func (e *Exporter) rewriteFixedTotals(f *excelize.File) error {
	wh, err := sumWholesaleRetail(f, "批发")
	if err != nil {
		return err
	}
	re, err := sumWholesaleRetail(f, "零售")
	if err != nil {
		return err
	}
	acc, err := sumAccommodationCatering(f, "住宿")
	if err != nil {
		return err
	}
	cat, err := sumAccommodationCatering(f, "餐饮")
	if err != nil {
		return err
	}

	if err := rewriteTotalsWholesaleRetail(f, "批发", wh); err != nil {
		return err
	}
	if err := rewriteTotalsWholesaleRetail(f, "零售", re); err != nil {
		return err
	}
	if err := rewriteTotalsAccommodationCatering(f, "住宿", acc); err != nil {
		return err
	}
	if err := rewriteTotalsAccommodationCatering(f, "餐饮", cat); err != nil {
		return err
	}

	if err := rewriteOverallRetailAreaOnWholesale(f, wh, re, acc, cat); err != nil {
		return err
	}
	if err := rewriteFixedSummarySheet(f, wh, re, acc, cat); err != nil {
		return err
	}

	return nil
}

// ---------- 行写入：批零（批发/零售/批零总表） ----------

func writeWRRowAt(f *excelize.File, sheet string, row int, r *model.WholesaleRetail) error {
	creditCode := strings.TrimSpace(r.CreditCode)
	name := strings.TrimSpace(r.Name)
	if creditCode == "" || name == "" {
		return fmt.Errorf("%s 第 %d 行企业信息为空（统一社会信用代码/单位详细名称）", sheet, row)
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("A%d", row), creditCode); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("B%d", row), name); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("D%d", row), r.SalesCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("G%d", row), r.SalesCurrentCumulative); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("J%d", row), r.RetailCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("M%d", row), r.RetailCurrentCumulative); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("F%d", row), ratePercent(r.SalesCurrentMonth, r.SalesLastYearMonth)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("I%d", row), ratePercent(r.SalesCurrentCumulative, r.SalesLastYearCumulative)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("L%d", row), ratePercent(r.RetailCurrentMonth, r.RetailLastYearMonth)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("O%d", row), ratePercent(r.RetailCurrentCumulative, r.RetailLastYearCumulative)); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("P%d", row), r.FirstReportIP); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("Q%d", row), r.FillIP); err != nil {
		return err
	}

	return nil
}

// ---------- 行写入：住餐（住宿/餐饮） ----------

func writeACRowAt(
	f *excelize.File,
	sheet string,
	row int,
	r *model.AccommodationCatering,
	retailCur float64,
	retailLast float64,
	retailCurCum float64,
	retailLastCum float64,
) error {
	creditCode := strings.TrimSpace(r.CreditCode)
	name := strings.TrimSpace(r.Name)
	if creditCode == "" || name == "" {
		return fmt.Errorf("%s 第 %d 行企业信息为空（统一社会信用代码/单位详细名称）", sheet, row)
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("A%d", row), creditCode); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("B%d", row), name); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("D%d", row), r.RevenueCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("G%d", row), r.RevenueCurrentCumulative); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("F%d", row), ratePercent(r.RevenueCurrentMonth, r.RevenueLastYearMonth)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("I%d", row), ratePercent(r.RevenueCurrentCumulative, r.RevenueLastYearCumulative)); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("J%d", row), r.RoomCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("L%d", row), r.RoomCurrentCumulative); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("N%d", row), r.FoodCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("P%d", row), r.FoodCurrentCumulative); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("R%d", row), r.GoodsCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("T%d", row), r.GoodsCurrentCumulative); err != nil {
		return err
	}

	// 模板右侧 V-Y：衍生“零售额”（餐费 + 商品销售）
	if err := setCellValue(f, sheet, fmt.Sprintf("V%d", row), retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("X%d", row), retailCurCum); err != nil {
		return err
	}
	// W/Y（上年）模板中已有历史值；为保证一致性与稳定定位，这里也回写一次
	if err := setCellValue(f, sheet, fmt.Sprintf("W%d", row), retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("Y%d", row), retailLastCum); err != nil {
		return err
	}

	return nil
}

func fillWRSheetByIndustryCodeOrder(f *excelize.File, sheet string, records []*model.WholesaleRetail) error {
	maxRow, err := findMaxDataRow(f, sheet, "C", 2)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("%s 没有可用数据记录", sheet)
	}

	byCode := map[string][]*model.WholesaleRetail{}
	for _, r := range records {
		code := normalizeCodeText(r.IndustryCode)
		byCode[code] = append(byCode[code], r)
	}
	for k := range byCode {
		rs := byCode[k]
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].RowNo != rs[j].RowNo {
				return rs[i].RowNo < rs[j].RowNo
			}
			return rs[i].ID < rs[j].ID
		})
		byCode[k] = rs
	}

	next := map[string]int{}
	for row := 2; row <= maxRow; row++ {
		code, err := getCellString(f, sheet, fmt.Sprintf("C%d", row))
		if err != nil {
			return err
		}
		codeKey := normalizeCodeText(code)
		list := byCode[codeKey]
		i := next[codeKey]
		if i >= len(list) {
			return fmt.Errorf("%s 第 %d 行无法匹配到企业记录（行业代码=%s）", sheet, row, strings.TrimSpace(code))
		}
		next[codeKey] = i + 1
		if err := writeWRRowAt(f, sheet, row, list[i]); err != nil {
			return err
		}
	}
	return nil
}

func fillACIndustrySheetByIndustryCodeOrder(f *excelize.File, sheet string, records []*model.AccommodationCatering) error {
	maxRow, err := findMaxDataRow(f, sheet, "C", 2)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("%s 没有可用数据记录", sheet)
	}

	byCode := map[string][]*model.AccommodationCatering{}
	for _, r := range records {
		code := normalizeCodeText(r.IndustryCode)
		byCode[code] = append(byCode[code], r)
	}
	for k := range byCode {
		rs := byCode[k]
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].RowNo != rs[j].RowNo {
				return rs[i].RowNo < rs[j].RowNo
			}
			return rs[i].ID < rs[j].ID
		})
		byCode[k] = rs
	}

	next := map[string]int{}
	for row := 2; row <= maxRow; row++ {
		code, err := getCellString(f, sheet, fmt.Sprintf("C%d", row))
		if err != nil {
			return err
		}
		codeKey := normalizeCodeText(code)
		list := byCode[codeKey]
		i := next[codeKey]
		if i >= len(list) {
			return fmt.Errorf("%s 第 %d 行无法匹配到企业记录（行业代码=%s）", sheet, row, strings.TrimSpace(code))
		}
		next[codeKey] = i + 1
		r := list[i]
		retailCur := r.FoodCurrentMonth + r.GoodsCurrentMonth
		retailLast := r.FoodLastYearMonth + r.GoodsLastYearMonth
		retailCurCum := r.FoodCurrentCumulative + r.GoodsCurrentCumulative
		retailLastCum := r.FoodLastYearCumulative + r.GoodsLastYearCumulative
		if err := writeACRowAt(f, sheet, row, r, retailCur, retailLast, retailCurCum, retailLastCum); err != nil {
			return err
		}
	}
	return nil
}

func fillACTotalSheetByIndustryCodeOrder(f *excelize.File, sheet string, records []*model.AccommodationCatering) error {
	maxRow, err := findMaxDataRow(f, sheet, "C", 2)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("%s 没有可用数据记录", sheet)
	}

	byCode := map[string][]*model.AccommodationCatering{}
	for _, r := range records {
		code := normalizeCodeText(r.IndustryCode)
		byCode[code] = append(byCode[code], r)
	}
	for k := range byCode {
		rs := byCode[k]
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].RowNo != rs[j].RowNo {
				return rs[i].RowNo < rs[j].RowNo
			}
			return rs[i].ID < rs[j].ID
		})
		byCode[k] = rs
	}

	next := map[string]int{}
	for row := 2; row <= maxRow; row++ {
		code, err := getCellString(f, sheet, fmt.Sprintf("C%d", row))
		if err != nil {
			return err
		}
		codeKey := normalizeCodeText(code)
		list := byCode[codeKey]
		i := next[codeKey]
		if i >= len(list) {
			return fmt.Errorf("%s 第 %d 行无法匹配到企业记录（行业代码=%s）", sheet, row, strings.TrimSpace(code))
		}
		next[codeKey] = i + 1
		if err := writeACTotalRowAt(f, sheet, row, list[i]); err != nil {
			return err
		}
	}
	return nil
}

func writeACTotalRowAt(f *excelize.File, sheet string, row int, r *model.AccommodationCatering) error {
	creditCode := strings.TrimSpace(r.CreditCode)
	name := strings.TrimSpace(r.Name)
	if creditCode == "" || name == "" {
		return fmt.Errorf("%s 第 %d 行企业信息为空（统一社会信用代码/单位详细名称）", sheet, row)
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("A%d", row), creditCode); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("B%d", row), name); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("D%d", row), r.RevenueCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("G%d", row), r.RevenueCurrentCumulative); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("F%d", row), ratePercent(r.RevenueCurrentMonth, r.RevenueLastYearMonth)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("I%d", row), ratePercent(r.RevenueCurrentCumulative, r.RevenueLastYearCumulative)); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("J%d", row), r.RoomCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("L%d", row), r.RoomCurrentCumulative); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("N%d", row), r.FoodCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("P%d", row), r.FoodCurrentCumulative); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("R%d", row), r.GoodsCurrentMonth); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("T%d", row), r.GoodsCurrentCumulative); err != nil {
		return err
	}

	return nil
}

func normalizeCodeText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, ",", "")
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		if math.Abs(v-math.Round(v)) <= 1e-9 {
			return strconv.FormatInt(int64(math.Round(v)), 10)
		}
	}
	return s
}

// ---------- 汇总区重写 ----------

func sumWholesaleRetail(f *excelize.File, sheet string) (wrSums, error) {
	maxRow, err := findMaxDataRow(f, sheet, "C", 2)
	if err != nil {
		return wrSums{}, err
	}
	s := wrSums{maxRow: maxRow}
	for row := 2; row <= maxRow; row++ {
		d, _ := getCellFloat(f, sheet, fmt.Sprintf("D%d", row))
		e, _ := getCellFloat(f, sheet, fmt.Sprintf("E%d", row))
		g, _ := getCellFloat(f, sheet, fmt.Sprintf("G%d", row))
		h, _ := getCellFloat(f, sheet, fmt.Sprintf("H%d", row))
		j, _ := getCellFloat(f, sheet, fmt.Sprintf("J%d", row))
		k, _ := getCellFloat(f, sheet, fmt.Sprintf("K%d", row))
		m, _ := getCellFloat(f, sheet, fmt.Sprintf("M%d", row))
		n, _ := getCellFloat(f, sheet, fmt.Sprintf("N%d", row))
		s.salesCur += d
		s.salesLast += e
		s.salesCurCum += g
		s.salesLastCum += h
		s.retailCur += j
		s.retailLast += k
		s.retailCurCum += m
		s.retailLastCum += n
	}
	return s, nil
}

func sumAccommodationCatering(f *excelize.File, sheet string) (wrSums, error) {
	maxRow, err := findMaxDataRow(f, sheet, "C", 2)
	if err != nil {
		return wrSums{}, err
	}
	s := wrSums{maxRow: maxRow}
	for row := 2; row <= maxRow; row++ {
		d, _ := getCellFloat(f, sheet, fmt.Sprintf("D%d", row))
		e, _ := getCellFloat(f, sheet, fmt.Sprintf("E%d", row))
		g, _ := getCellFloat(f, sheet, fmt.Sprintf("G%d", row))
		h, _ := getCellFloat(f, sheet, fmt.Sprintf("H%d", row))
		v, _ := getCellFloat(f, sheet, fmt.Sprintf("V%d", row))
		w, _ := getCellFloat(f, sheet, fmt.Sprintf("W%d", row))
		x, _ := getCellFloat(f, sheet, fmt.Sprintf("X%d", row))
		y, _ := getCellFloat(f, sheet, fmt.Sprintf("Y%d", row))
		s.salesCur += d
		s.salesLast += e
		s.salesCurCum += g
		s.salesLastCum += h
		s.retailCur += v
		s.retailLast += w
		s.retailCurCum += x
		s.retailLastCum += y
	}
	return s, nil
}

func rewriteTotalsWholesaleRetail(f *excelize.File, sheet string, sums wrSums) error {
	sumRow := sums.maxRow + 1
	growthRow := sums.maxRow + 2

	if err := setCellValue(f, sheet, fmt.Sprintf("D%d", sumRow), sums.salesCur); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("E%d", sumRow), sums.salesLast); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("G%d", sumRow), sums.salesCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("H%d", sumRow), sums.salesLastCum); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("J%d", sumRow), sums.retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("K%d", sumRow), sums.retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("M%d", sumRow), sums.retailCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("N%d", sumRow), sums.retailLastCum); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("E%d", growthRow), ratePercent(sums.salesCur, sums.salesLast)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("H%d", growthRow), ratePercent(sums.salesCurCum, sums.salesLastCum)); err != nil {
		return err
	}

	return nil
}

func rewriteTotalsAccommodationCatering(f *excelize.File, sheet string, sums wrSums) error {
	sumRow := sums.maxRow + 1
	growthRow := sums.maxRow + 2

	if err := setCellValue(f, sheet, fmt.Sprintf("D%d", sumRow), sums.salesCur); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("E%d", sumRow), sums.salesLast); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("G%d", sumRow), sums.salesCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("H%d", sumRow), sums.salesLastCum); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("V%d", sumRow), sums.retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("W%d", sumRow), sums.retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("X%d", sumRow), sums.retailCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("Y%d", sumRow), sums.retailLastCum); err != nil {
		return err
	}

	if err := setCellValue(f, sheet, fmt.Sprintf("E%d", growthRow), ratePercent(sums.salesCur, sums.salesLast)); err != nil {
		return err
	}
	if err := setCellValue(f, sheet, fmt.Sprintf("H%d", growthRow), ratePercent(sums.salesCurCum, sums.salesLastCum)); err != nil {
		return err
	}

	return nil
}

func rewriteOverallRetailAreaOnWholesale(f *excelize.File, wh, re, acc, cat wrSums) error {
	ws := "批发"
	whMax := wh.maxRow
	sumRow := whMax + 1
	growthRow := whMax + 2
	accRow := growthRow + 1
	catRow := growthRow + 2
	totalRow := growthRow + 3
	totalGrowthRow := growthRow + 4

	if err := setCellValue(f, ws, fmt.Sprintf("J%d", growthRow), re.retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("K%d", growthRow), re.retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("M%d", growthRow), re.retailCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("N%d", growthRow), re.retailLastCum); err != nil {
		return err
	}

	if err := setCellValue(f, ws, fmt.Sprintf("J%d", accRow), acc.retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("K%d", accRow), acc.retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("M%d", accRow), acc.retailCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("N%d", accRow), acc.retailLastCum); err != nil {
		return err
	}

	if err := setCellValue(f, ws, fmt.Sprintf("J%d", catRow), cat.retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("K%d", catRow), cat.retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("M%d", catRow), cat.retailCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("N%d", catRow), cat.retailLastCum); err != nil {
		return err
	}

	overallCur := wh.retailCur + re.retailCur + acc.retailCur + cat.retailCur
	overallLast := wh.retailLast + re.retailLast + acc.retailLast + cat.retailLast
	overallCurCum := wh.retailCurCum + re.retailCurCum + acc.retailCurCum + cat.retailCurCum
	overallLastCum := wh.retailLastCum + re.retailLastCum + acc.retailLastCum + cat.retailLastCum

	if err := setCellValue(f, ws, fmt.Sprintf("J%d", totalRow), overallCur); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("K%d", totalRow), overallLast); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("M%d", totalRow), overallCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("N%d", totalRow), overallLastCum); err != nil {
		return err
	}

	if err := setCellValue(f, ws, fmt.Sprintf("K%d", totalGrowthRow), ratePercent(overallCur, overallLast)); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("N%d", totalGrowthRow), ratePercent(overallCurCum, overallLastCum)); err != nil {
		return err
	}

	// 回写批发自身零售额到汇总行（保证模板 key 与取数一致）
	if err := setCellValue(f, ws, fmt.Sprintf("J%d", sumRow), wh.retailCur); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("K%d", sumRow), wh.retailLast); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("M%d", sumRow), wh.retailCurCum); err != nil {
		return err
	}
	if err := setCellValue(f, ws, fmt.Sprintf("N%d", sumRow), wh.retailLastCum); err != nil {
		return err
	}

	return nil
}

func rewriteFixedSummarySheet(f *excelize.File, wh, re, acc, cat wrSums) error {
	summary := "汇总表（定）"
	ws := "批发"
	whMax := wh.maxRow
	overallRow := whMax + 5

	overallCur, _ := getCellFloat(f, ws, fmt.Sprintf("J%d", overallRow))
	overallLast, _ := getCellFloat(f, ws, fmt.Sprintf("K%d", overallRow))
	overallCurCum, _ := getCellFloat(f, ws, fmt.Sprintf("M%d", overallRow))
	overallLastCum, _ := getCellFloat(f, ws, fmt.Sprintf("N%d", overallRow))

	if err := setCellValue(f, summary, "G4", overallCur/10.0); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "H4", overallLast/10.0); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "I4", overallCurCum/10.0); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "J4", overallLastCum/10.0); err != nil {
		return err
	}

	whGrowthMonth, _ := getCellFloat(f, "批发", fmt.Sprintf("E%d", whMax+2))
	whGrowthCum, _ := getCellFloat(f, "批发", fmt.Sprintf("H%d", whMax+2))
	reGrowthMonth, _ := getCellFloat(f, "零售", fmt.Sprintf("E%d", re.maxRow+2))
	reGrowthCum, _ := getCellFloat(f, "零售", fmt.Sprintf("H%d", re.maxRow+2))
	accGrowthMonth, _ := getCellFloat(f, "住宿", fmt.Sprintf("E%d", acc.maxRow+2))
	accGrowthCum, _ := getCellFloat(f, "住宿", fmt.Sprintf("H%d", acc.maxRow+2))
	catGrowthMonth, _ := getCellFloat(f, "餐饮", fmt.Sprintf("E%d", cat.maxRow+2))
	catGrowthCum, _ := getCellFloat(f, "餐饮", fmt.Sprintf("H%d", cat.maxRow+2))

	if err := setCellValue(f, summary, "K4", roundHalfUp(whGrowthMonth, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "L4", roundHalfUp(whGrowthCum, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "M4", roundHalfUp(reGrowthMonth, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "N4", roundHalfUp(reGrowthCum, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "O4", roundHalfUp(accGrowthMonth, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "P4", roundHalfUp(accGrowthCum, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "Q4", roundHalfUp(catGrowthMonth, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "R4", roundHalfUp(catGrowthCum, 1)); err != nil {
		return err
	}

	monthRate := -100.0
	if overallLast != 0 {
		monthRate = (overallCur/overallLast - 1) * 100.0
	}
	cumRate := -100.0
	if overallLastCum != 0 {
		cumRate = (overallCurCum/overallLastCum - 1) * 100.0
	}

	if err := setCellValue(f, summary, "S4", roundHalfUp(monthRate, 1)); err != nil {
		return err
	}
	if err := setCellValue(f, summary, "T4", roundHalfUp(cumRate, 1)); err != nil {
		return err
	}

	return nil
}

// ---------- 通用工具函数 ----------

func findMaxDataRow(f *excelize.File, sheet, col string, startRow int) (int, error) {
	for r := startRow; r <= 50000; r++ {
		v, err := getCellString(f, sheet, fmt.Sprintf("%s%d", col, r))
		if err != nil {
			return 0, err
		}
		if strings.TrimSpace(v) == "" {
			if r == startRow {
				return 0, fmt.Errorf("%s 没有数据行", sheet)
			}
			return r - 1, nil
		}
	}
	return 0, fmt.Errorf("%s 数据行过多，超出扫描上限", sheet)
}

func getCellString(f *excelize.File, sheet, cell string) (string, error) {
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return "", err
	}
	return v, nil
}

func getCellFloat(f *excelize.File, sheet, cell string) (float64, error) {
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return 0, err
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
	v = strings.ReplaceAll(v, ",", "")
	val, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, nil
	}
	return val, nil
}

func setCellValue(f *excelize.File, sheet, cell string, value interface{}) error {
	return f.SetCellValue(sheet, cell, value)
}

func roundHalfUp(v float64, digits int) float64 {
	if digits < 0 {
		return v
	}
	scale := math.Pow10(digits)
	x := v * scale
	if x >= 0 {
		return math.Floor(x+0.5) / scale
	}
	return -math.Floor(-x+0.5) / scale
}

func ratePercent(cur, last float64) float64 {
	if last == 0 {
		return -100.0
	}
	return roundHalfUp((cur/last-1.0)*100.0, 2)
}
