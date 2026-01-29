package excel

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
)

// Parser Excel解析器
type Parser struct {
	file    *excelize.File
	fileID  string
	mapping *model.FieldMapping
}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{
		fileID: uuid.New().String(),
	}
}

// LoadFile 加载Excel文件
func (p *Parser) LoadFile(reader io.Reader) error {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return fmt.Errorf("failed to open excel: %w", err)
	}
	p.file = file
	return nil
}

// GetFileID 获取文件ID
func (p *Parser) GetFileID() string {
	return p.fileID
}

// Workbook 返回已加载的工作簿对象（只读使用）
func (p *Parser) Workbook() *excelize.File {
	return p.file
}

// GetSheets 获取工作表列表
func (p *Parser) GetSheets() ([]model.SheetInfo, error) {
	if p.file == nil {
		return nil, errors.New("no file loaded")
	}

	sheets := p.file.GetSheetList()
	result := make([]model.SheetInfo, 0, len(sheets))

	for _, name := range sheets {
		rows, err := p.file.GetRows(name)
		if err != nil {
			continue
		}
		result = append(result, model.SheetInfo{
			Name:     name,
			RowCount: len(rows),
		})
	}

	return result, nil
}

// GetColumns 获取列名
func (p *Parser) GetColumns(sheet string) ([]string, error) {
	if p.file == nil {
		return nil, errors.New("no file loaded")
	}

	rows, err := p.file.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.New("empty sheet")
	}

	return rows[0], nil
}

// GetPreviewRows 获取预览行
func (p *Parser) GetPreviewRows(sheet string, limit int) ([][]string, error) {
	if p.file == nil {
		return nil, errors.New("no file loaded")
	}

	rows, err := p.file.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	if len(rows) <= 1 {
		return [][]string{}, nil
	}

	end := limit + 1
	if end > len(rows) {
		end = len(rows)
	}

	return rows[1:end], nil
}

// SetMapping 设置字段映射
func (p *Parser) SetMapping(mapping *model.FieldMapping) {
	p.mapping = mapping
}

// Parse 解析企业数据
func (p *Parser) Parse(sheet string) ([]*model.Company, error) {
	if p.file == nil {
		return nil, errors.New("no file loaded")
	}
	if p.mapping == nil {
		return nil, errors.New("no mapping configured")
	}

	rows, err := p.file.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	if len(rows) <= 1 {
		return []*model.Company{}, nil
	}

	// 构建列名到索引的映射
	header := rows[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[col] = i
	}

	companies := make([]*model.Company, 0, len(rows)-1)

	for i, row := range rows[1:] {
		company, err := p.parseRow(row, colIndex, i+2)
		if err != nil {
			continue // 跳过错误行
		}
		companies = append(companies, company)
	}

	return companies, nil
}

// parseRow 解析单行数据
func (p *Parser) parseRow(row []string, colIndex map[string]int, rowNum int) (*model.Company, error) {
	getValue := func(field string) string {
		if idx, ok := colIndex[field]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	getFloat := func(field string) float64 {
		val := getValue(field)
		if val == "" {
			return 0
		}
		// 移除千分位分隔符
		val = strings.ReplaceAll(val, ",", "")
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}

	getInt := func(field string) int {
		val := getValue(field)
		if val == "" {
			return 0
		}
		i, _ := strconv.Atoi(val)
		return i
	}

	name := getValue(p.mapping.CompanyName)
	if name == "" {
		return nil, fmt.Errorf("row %d: empty company name", rowNum)
	}

	company := &model.Company{
		ID:                       uuid.New().String(),
		Name:                     name,
		CreditCode:               getValue(p.mapping.CreditCode),
		IndustryCode:             getValue(p.mapping.IndustryCode),
		CompanyScale:             getInt(p.mapping.CompanyScale),
		RetailCurrentMonth:       getFloat(p.mapping.RetailCurrentMonth),
		RetailLastYearMonth:      getFloat(p.mapping.RetailLastYearMonth),
		RetailCurrentCumulative:  getFloat(p.mapping.RetailCurrentCumulative),
		RetailLastYearCumulative: getFloat(p.mapping.RetailLastYearCumulative),
		SalesCurrentMonth:        getFloat(p.mapping.SalesCurrentMonth),
		SalesLastYearMonth:       getFloat(p.mapping.SalesLastYearMonth),
		SalesCurrentCumulative:   getFloat(p.mapping.SalesCurrentCumulative),
		SalesLastYearCumulative:  getFloat(p.mapping.SalesLastYearCumulative),
	}

	// 根据行业代码判断行业类型
	company.IndustryType = detectIndustryType(company.IndustryCode)
	// 判断是否属于吃穿用类
	company.IsEatWearUse = isEatWearUse(company.IndustryCode)

	// 记录 Excel 原始值：用于“重置为原始导入数据”
	company.OriginalInitialized = true
	company.OriginalName = company.Name
	company.OriginalRetailLastYearMonth = company.RetailLastYearMonth
	company.OriginalRetailCurrentMonth = company.RetailCurrentMonth
	company.OriginalRetailLastYearCumulative = company.RetailLastYearCumulative
	company.OriginalRetailCurrentCumulative = company.RetailCurrentCumulative
	company.OriginalSalesLastYearMonth = company.SalesLastYearMonth
	company.OriginalSalesCurrentMonth = company.SalesCurrentMonth
	company.OriginalSalesLastYearCumulative = company.SalesLastYearCumulative
	company.OriginalSalesCurrentCumulative = company.SalesCurrentCumulative

	return company, nil
}

// detectIndustryType 根据行业代码判断行业类型
func detectIndustryType(code string) model.IndustryType {
	if len(code) < 2 {
		return model.IndustryRetail
	}

	prefix := code[:2]
	switch prefix {
	case "51": // 批发业
		return model.IndustryWholesale
	case "52": // 零售业
		return model.IndustryRetail
	case "61": // 住宿业
		return model.IndustryAccommodation
	case "62": // 餐饮业
		return model.IndustryCatering
	default:
		return model.IndustryRetail
	}
}

// isEatWearUse 判断是否属于吃穿用类
func isEatWearUse(code string) bool {
	// 吃穿用行业代码列表 (简化版)
	eatWearUseCodes := []string{
		"5211", "5212", "5213", // 食品零售
		"5221", "5222", "5223", // 服装零售
		"5241", "5242", "5243", // 日用品零售
		"5122", "5123", // 食品批发
	}

	for _, c := range eatWearUseCodes {
		if strings.HasPrefix(code, c) {
			return true
		}
	}
	return false
}

// Close 关闭文件
func (p *Parser) Close() error {
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

// ParseOptions 解析 Excel 为统一口径数据的选项
type ParseOptions struct {
	Month     int
	Overrides map[string]model.SheetType
}

// ParseCanonical 将输入工作簿解析为统一口径数据（先覆盖批零主表最小链路）
func ParseCanonical(wb *excelize.File, opts ParseOptions) (*model.CanonicalWorkbookData, error) {
	if wb == nil {
		return nil, errors.New("workbook is nil")
	}
	if opts.Month < 1 || opts.Month > 12 {
		return nil, errors.New("month must be 1-12")
	}

	resolved := ResolveWorkbook(wb, ResolveOptions{
		Month:     opts.Month,
		Overrides: opts.Overrides,
	})

	companies := make([]*model.CanonicalCompany, 0)
	mainOrder := []model.SheetType{
		model.SheetTypeWholesaleMain,
		model.SheetTypeRetailMain,
		model.SheetTypeAccommodationMain,
		model.SheetTypeCateringMain,
	}
	for _, t := range mainOrder {
		sheetName := resolved.MainSheets[t]
		if strings.TrimSpace(sheetName) == "" {
			continue
		}
		switch t {
		case model.SheetTypeWholesaleMain, model.SheetTypeRetailMain:
			companies = append(companies, parseWholesaleRetailMainSheetToCanonical(wb, sheetName, opts.Month)...)
		case model.SheetTypeAccommodationMain, model.SheetTypeCateringMain:
			companies = append(companies, parseAccommodationCateringMainSheetToCanonical(wb, sheetName, opts.Month)...)
		default:
			continue
		}
	}

	return &model.CanonicalWorkbookData{
		Month:     opts.Month,
		Companies: companies,
	}, nil
}

var (
	reMainYearMonthSales  = regexp.MustCompile(`^(\d{4})年(\d{1,2})月销售额$`)
	reMainYearMonthRetail = regexp.MustCompile(`^(\d{4})年(\d{1,2})月零售额$`)

	reMainYearRangeSales  = regexp.MustCompile(`^(\d{4})年1-(\d{1,2})月销售额$`)
	reMainYearRangeRetail = regexp.MustCompile(`^(\d{4})年1-(\d{1,2})月零售额$`)

	reMainLastYearSales       = regexp.MustCompile(`^(\d{4})年;\s*(\d{1,2})月;商品销售额;千元$`)
	reMainLastYearRetail      = regexp.MustCompile(`^(\d{4})年;\s*(\d{1,2})月;商品零售额;千元$`)
	reMainLastYearRangeSales  = regexp.MustCompile(`^(\d{4})年;\s*1-(\d{1,2})月;商品销售额;千元$`)
	reMainLastYearRangeRetail = regexp.MustCompile(`^(\d{4})年;\s*1-(\d{1,2})月;商品零售额;千元$`)
)

func parseWholesaleRetailMainSheetToCanonical(wb *excelize.File, sheetName string, month int) []*model.CanonicalCompany {
	rows, err := wb.GetRows(sheetName)
	if err != nil || len(rows) <= 1 {
		return []*model.CanonicalCompany{}
	}

	header := rows[0]

	colCredit := findExactCol(header, "统一社会信用代码")
	colName := findExactCol(header, "单位详细名称")
	colIndustryCode := findIndustryCodeCol(header)
	colCompanyScale := findContainsCol(header, "单位规模")

	colSalesCur := findMainYearMonthCol(header, month, reMainYearMonthSales)
	colRetailCur := findMainYearMonthCol(header, month, reMainYearMonthRetail)
	colSalesCurCum := findMainYearRangeCol(header, month, reMainYearRangeSales)
	colRetailCurCum := findMainYearRangeCol(header, month, reMainYearRangeRetail)

	colSalesPrevCum := -1
	colRetailPrevCum := -1
	if month > 1 {
		colSalesPrevCum = findMainYearRangeCol(header, month-1, reMainYearRangeSales)
		colRetailPrevCum = findMainYearRangeCol(header, month-1, reMainYearRangeRetail)
	}

	colSalesLast := findMainYearMonthCol(header, month, reMainLastYearSales)
	colRetailLast := findMainYearMonthCol(header, month, reMainLastYearRetail)
	colSalesLastCum := findMainYearRangeCol(header, month, reMainLastYearRangeSales)
	colRetailLastCum := findMainYearRangeCol(header, month, reMainLastYearRangeRetail)

	out := make([]*model.CanonicalCompany, 0, len(rows)-1)
	for i, row := range rows[1:] {
		rowNo := i + 2
		credit := getCell(row, colCredit)
		name := getCell(row, colName)
		industryCode := strings.TrimSpace(getCell(row, colIndustryCode))
		// 预估版里“空白格式行”很多，必须严格跳过；只认行业代码存在的行。
		if industryCode == "" {
			continue
		}

		scale := parseInt(getCell(row, colCompanyScale))

		c := &model.CanonicalCompany{
			RowNo:        rowNo,
			CreditCode:   strings.TrimSpace(credit),
			Name:         strings.TrimSpace(name),
			IndustryCode: industryCode,
			IndustryType: detectIndustryType(industryCode),
			CompanyScale: scale,
			IsEatWearUse: isEatWearUse(industryCode),
		}

		salesCur, hasSalesCur := parseOptionalFloat(getCell(row, colSalesCur))
		retailCur, hasRetailCur := parseOptionalFloat(getCell(row, colRetailCur))

		salesCurCum := parseFloat(getCell(row, colSalesCurCum))
		retailCurCum := parseFloat(getCell(row, colRetailCurCum))

		// “预估”版常见：当月值为空，但同时提供“1-(m-1)”与“1-m”累计。此时按差值推导当月值。
		if !hasSalesCur && month > 1 && colSalesCurCum >= 0 && colSalesPrevCum >= 0 {
			salesCur = salesCurCum - parseFloat(getCell(row, colSalesPrevCum))
		}
		if !hasRetailCur && month > 1 && colRetailCurCum >= 0 && colRetailPrevCum >= 0 {
			retailCur = retailCurCum - parseFloat(getCell(row, colRetailPrevCum))
		}

		c.Sales = model.CanonicalAmount{
			CurrentMonth:       salesCur,
			LastYearMonth:      parseFloat(getCell(row, colSalesLast)),
			CurrentCumulative:  salesCurCum,
			LastYearCumulative: parseFloat(getCell(row, colSalesLastCum)),
		}
		c.Retail = model.CanonicalAmount{
			CurrentMonth:       retailCur,
			LastYearMonth:      parseFloat(getCell(row, colRetailLast)),
			CurrentCumulative:  retailCurCum,
			LastYearCumulative: parseFloat(getCell(row, colRetailLastCum)),
		}

		out = append(out, c)
	}

	return out
}

var (
	reMainYearMonthRevenue     = regexp.MustCompile(`^(\d{4})年(\d{1,2})月营业额$`)
	reMainYearRangeRevenue     = regexp.MustCompile(`^(\d{4})年1-(\d{1,2})月营业额$`)
	reMainLastYearMonthRevenue = regexp.MustCompile(`^(\d{4})年(\d{1,2})月;\s*营业额总计;千元$`)
	reMainLastYearRangeRevenue = regexp.MustCompile(`^(\d{4})年\s*1-(\d{1,2})月;\s*营业额总计;千元$`)

	reMainYearMonthRoom     = regexp.MustCompile(`^(\d{4})年(\d{1,2})月客房收入$`)
	reMainYearRangeRoom     = regexp.MustCompile(`^(\d{4})年1-(\d{1,2})月客房收入$`)
	reMainLastYearMonthRoom = regexp.MustCompile(`^(\d{4})年(\d{1,2})月;\s*营业额总计;客房收入;千元$`)
	reMainLastYearRangeRoom = regexp.MustCompile(`^(\d{4})年\s*1-(\d{1,2})月;\s*营业额总计;客房收入;千元$`)

	reMainYearMonthFood     = regexp.MustCompile(`^(\d{4})年(\d{1,2})月餐费收入$`)
	reMainYearRangeFood     = regexp.MustCompile(`^(\d{4})年1-(\d{1,2})月餐费收入$`)
	reMainLastYearMonthFood = regexp.MustCompile(`^(\d{4})年(\d{1,2})月;\s*营业额总计;餐费收入;千元$`)
	reMainLastYearRangeFood = regexp.MustCompile(`^(\d{4})年\s*1-(\d{1,2})月;\s*营业额总计;餐费收入;千元$`)

	reMainYearMonthGoods     = regexp.MustCompile(`^(\d{4})年(\d{1,2})月销售额$`)
	reMainLastYearMonthGoods = regexp.MustCompile(`^(\d{4})年(\d{1,2})月;\s*营业额总计;商品销售额;千元$`)
	reMainLastYearRangeGoods = regexp.MustCompile(`^(\d{4})年\s*1-(\d{1,2})月;\s*营业额总计;商品销售额;千元$`)
)

func parseAccommodationCateringMainSheetToCanonical(wb *excelize.File, sheetName string, month int) []*model.CanonicalCompany {
	rows, err := wb.GetRows(sheetName)
	if err != nil || len(rows) <= 1 {
		return []*model.CanonicalCompany{}
	}

	header := rows[0]

	colCredit := findExactCol(header, "统一社会信用代码")
	colName := findExactCol(header, "单位详细名称")
	colIndustryCode := findIndustryCodeCol(header)

	// 住餐 sheet 的“单位规模”可能不存在；存在就读。
	colCompanyScale := findContainsCol(header, "单位规模")

	colRevCur := findMainYearMonthCol(header, month, reMainYearMonthRevenue)
	colRevCurCum := findMainYearRangeCol(header, month, reMainYearRangeRevenue)
	colRevLast := findMainYearMonthCol(header, month, reMainLastYearMonthRevenue)
	colRevLastCum := findMainYearRangeCol(header, month, reMainLastYearRangeRevenue)
	colRevPrevCum := -1
	if month > 1 {
		colRevPrevCum = findMainYearRangeCol(header, month-1, reMainYearRangeRevenue)
	}

	colRoomCur := findMainYearMonthCol(header, month, reMainYearMonthRoom)
	colRoomCurCum := findMainYearRangeCol(header, month, reMainYearRangeRoom)
	colRoomLast := findMainYearMonthCol(header, month, reMainLastYearMonthRoom)
	colRoomLastCum := findMainYearRangeCol(header, month, reMainLastYearRangeRoom)
	colRoomPrevCum := -1
	if month > 1 {
		colRoomPrevCum = findMainYearRangeCol(header, month-1, reMainYearRangeRoom)
	}

	colFoodCur := findMainYearMonthCol(header, month, reMainYearMonthFood)
	colFoodLast := findMainYearMonthCol(header, month, reMainLastYearMonthFood)
	colFoodLastCum := findMainYearRangeCol(header, month, reMainLastYearRangeFood)
	colFoodPrevCum := -1
	if month > 1 {
		colFoodPrevCum = findMainYearRangeCol(header, month-1, reMainYearRangeFood)
	}
	// 预估版里常见“1-12月餐费收入”（无年份），作为兜底读作本年累计
	colFoodCurCum := findExactCol(header, fmt.Sprintf("1-%d月餐费收入", month))
	if colFoodCurCum < 0 {
		colFoodCurCum = findContainsCol(header, "1-12月餐费收入")
	}

	colGoodsCur := findMainYearMonthCol(header, month, reMainYearMonthGoods)
	colGoodsLast := findMainYearMonthCol(header, month, reMainLastYearMonthGoods)
	colGoodsLastCum := findMainYearRangeCol(header, month, reMainLastYearRangeGoods)
	colGoodsPrevCum := -1
	if month > 1 {
		// “2025年1-11月销售额”这类列名与批零主表一致，复用 reMainYearRangeSales
		colGoodsPrevCum = findMainYearRangeCol(header, month-1, reMainYearRangeSales)
	}
	colGoodsCurCum := findExactCol(header, fmt.Sprintf("1-%d月销售额", month))
	if colGoodsCurCum < 0 {
		colGoodsCurCum = findContainsCol(header, "1-12月销售额")
	}

	out := make([]*model.CanonicalCompany, 0, len(rows)-1)
	for i, row := range rows[1:] {
		rowNo := i + 2
		industryCode := strings.TrimSpace(getCell(row, colIndustryCode))
		if industryCode == "" {
			continue
		}

		scale := parseInt(getCell(row, colCompanyScale))
		c := &model.CanonicalCompany{
			RowNo:        rowNo,
			CreditCode:   strings.TrimSpace(getCell(row, colCredit)),
			Name:         strings.TrimSpace(getCell(row, colName)),
			IndustryCode: industryCode,
			IndustryType: detectIndustryType(industryCode),
			CompanyScale: scale,
		}

		revCur, hasRevCur := parseOptionalFloat(getCell(row, colRevCur))
		revCurCum := parseFloat(getCell(row, colRevCurCum))
		if !hasRevCur && month > 1 && colRevCurCum >= 0 && colRevPrevCum >= 0 {
			revCur = revCurCum - parseFloat(getCell(row, colRevPrevCum))
		}

		roomCur, hasRoomCur := parseOptionalFloat(getCell(row, colRoomCur))
		roomCurCum := parseFloat(getCell(row, colRoomCurCum))
		if !hasRoomCur && month > 1 && colRoomCurCum >= 0 && colRoomPrevCum >= 0 {
			roomCur = roomCurCum - parseFloat(getCell(row, colRoomPrevCum))
		}

		foodCur, hasFoodCur := parseOptionalFloat(getCell(row, colFoodCur))
		foodCurCum := parseFloat(getCell(row, colFoodCurCum))
		if !hasFoodCur && month > 1 && colFoodCurCum >= 0 && colFoodPrevCum >= 0 {
			foodCur = foodCurCum - parseFloat(getCell(row, colFoodPrevCum))
		}

		goodsCur, hasGoodsCur := parseOptionalFloat(getCell(row, colGoodsCur))
		goodsCurCum := parseFloat(getCell(row, colGoodsCurCum))
		if !hasGoodsCur && month > 1 && colGoodsCurCum >= 0 && colGoodsPrevCum >= 0 {
			goodsCur = goodsCurCum - parseFloat(getCell(row, colGoodsPrevCum))
		}

		c.Revenue = model.CanonicalAmount{
			CurrentMonth:       revCur,
			LastYearMonth:      parseFloat(getCell(row, colRevLast)),
			CurrentCumulative:  revCurCum,
			LastYearCumulative: parseFloat(getCell(row, colRevLastCum)),
		}
		c.RoomRevenue = model.CanonicalAmount{
			CurrentMonth:       roomCur,
			LastYearMonth:      parseFloat(getCell(row, colRoomLast)),
			CurrentCumulative:  roomCurCum,
			LastYearCumulative: parseFloat(getCell(row, colRoomLastCum)),
		}
		c.FoodRevenue = model.CanonicalAmount{
			CurrentMonth:       foodCur,
			LastYearMonth:      parseFloat(getCell(row, colFoodLast)),
			CurrentCumulative:  foodCurCum,
			LastYearCumulative: parseFloat(getCell(row, colFoodLastCum)),
		}
		c.GoodsSales = model.CanonicalAmount{
			CurrentMonth:       goodsCur,
			LastYearMonth:      parseFloat(getCell(row, colGoodsLast)),
			CurrentCumulative:  goodsCurCum,
			LastYearCumulative: parseFloat(getCell(row, colGoodsLastCum)),
		}

		out = append(out, c)
	}

	return out
}

func findExactCol(headers []string, want string) int {
	for i, h := range headers {
		if strings.TrimSpace(h) == want {
			return i
		}
	}
	return -1
}

func findContainsCol(headers []string, sub string) int {
	for i, h := range headers {
		if strings.Contains(strings.TrimSpace(h), sub) {
			return i
		}
	}
	return -1
}

func findIndustryCodeCol(headers []string) int {
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if strings.Contains(h, "行业代码") && strings.Contains(h, "4754") {
			return i
		}
	}
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if strings.Contains(h, "行业代码") {
			return i
		}
	}
	return -1
}

func findMainYearMonthCol(headers []string, month int, re *regexp.Regexp) int {
	bestCol := -1
	bestYear := -1
	for i, h := range headers {
		h = strings.TrimSpace(h)
		m := re.FindStringSubmatch(h)
		if len(m) != 3 {
			continue
		}
		y := parseInt(m[1])
		mm := parseInt(m[2])
		if mm != month || y <= 0 {
			continue
		}
		if y > bestYear {
			bestYear = y
			bestCol = i
		}
	}
	return bestCol
}

func findMainYearRangeCol(headers []string, month int, re *regexp.Regexp) int {
	bestCol := -1
	bestYear := -1
	for i, h := range headers {
		h = strings.TrimSpace(h)
		m := re.FindStringSubmatch(h)
		if len(m) != 3 {
			continue
		}
		y := parseInt(m[1])
		mm := parseInt(m[2])
		if mm != month || y <= 0 {
			continue
		}
		if y > bestYear {
			bestYear = y
			bestCol = i
		}
	}
	return bestCol
}

func getCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseOptionalFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	return parseFloat(s), true
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}
