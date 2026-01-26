package excel

import (
	"errors"
	"fmt"
	"io"
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
		"5122", "5123",         // 食品批发
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
