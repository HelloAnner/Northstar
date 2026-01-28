package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"northstar/internal/model"
	"northstar/internal/service/calculator"
	"northstar/internal/service/excel"
	"northstar/internal/service/project"
	"northstar/internal/service/store"
)

// Handlers API处理器
type Handlers struct {
	store     *store.MemoryStore
	engine    *calculator.Engine
	optimizer *calculator.Optimizer
	adjuster  *calculator.Adjuster
	projects  *project.Manager

	// 文件缓存
	parsers   map[string]*excel.Parser
	parsersMu sync.RWMutex

	uploads   map[string]*uploadedFile
	uploadsMu sync.RWMutex

	// 导出文件缓存
	exports   map[string]string
	exportsMu sync.RWMutex
}

type uploadedFile struct {
	FileName string
	Bytes    []byte
	Mapping  *model.FieldMapping
}

// NewHandlers 创建处理器
func NewHandlers(store *store.MemoryStore, engine *calculator.Engine, optimizer *calculator.Optimizer, projects *project.Manager) *Handlers {
	return &Handlers{
		store:     store,
		engine:    engine,
		optimizer: optimizer,
		adjuster:  calculator.NewAdjuster(store, engine),
		projects:  projects,
		parsers:   make(map[string]*excel.Parser),
		uploads:   make(map[string]*uploadedFile),
		exports:   make(map[string]string),
	}
}

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func errorResponse(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ==================== Projects ====================

// ListProjects 获取项目列表
func (h *Handlers) ListProjects(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}
	success(c, h.projects.ListProjects())
}

// CreateProject 新建项目（默认选中该项目）
func (h *Handlers) CreateProject(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	projectSummary, err := h.projects.CreateProject(strings.TrimSpace(req.Name))
	if err != nil {
		errorResponse(c, 5002, err.Error())
		return
	}

	success(c, projectSummary)
}

// SelectProject 选择/切换项目
func (h *Handlers) SelectProject(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}
	projectID := c.Param("projectId")
	projectSummary, err := h.projects.SelectProject(projectID)
	if err != nil {
		errorResponse(c, 5003, err.Error())
		return
	}
	success(c, projectSummary)
}

// GetCurrentProject 获取当前项目
func (h *Handlers) GetCurrentProject(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}
	cp, err := h.projects.Current()
	if err != nil {
		errorResponse(c, 5004, err.Error())
		return
	}
	success(c, cp)
}

// SaveCurrentProject 强制保存当前项目
func (h *Handlers) SaveCurrentProject(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}
	if err := h.projects.SaveNow(); err != nil {
		errorResponse(c, 5005, err.Error())
		return
	}
	success(c, gin.H{"saved": true})
}

// UndoCurrentProject 撤销上一次修改（恢复到上一次快照）
func (h *Handlers) UndoCurrentProject(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}

	if err := h.projects.UndoLast(); err != nil {
		errorResponse(c, 5008, err.Error())
		return
	}

	indicators := h.engine.Calculate()
	h.projects.ScheduleSave()

	success(c, gin.H{
		"indicators": indicators,
	})
}

// GetProjectDetail 获取项目详情
func (h *Handlers) GetProjectDetail(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}
	projectID := c.Param("projectId")
	detail, err := h.projects.GetProjectDetail(projectID)
	if err != nil {
		errorResponse(c, 5006, err.Error())
		return
	}
	success(c, detail)
}

// DeleteProject 删除项目（高危操作）
func (h *Handlers) DeleteProject(c *gin.Context) {
	if h.projects == nil {
		errorResponse(c, 5001, "项目管理不可用")
		return
	}
	projectID := c.Param("projectId")
	if err := h.projects.DeleteProject(projectID); err != nil {
		errorResponse(c, 5007, err.Error())
		return
	}
	success(c, gin.H{"deleted": true})
}

// UploadFile 上传Excel文件
func (h *Handlers) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		errorResponse(c, 1001, "请上传文件")
		return
	}
	defer file.Close()

	// 检查文件大小 (10MB)
	if header.Size > 10*1024*1024 {
		errorResponse(c, 1003, "文件过大，最大支持10MB")
		return
	}

	// 检查文件格式
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		errorResponse(c, 1002, "仅支持 .xlsx 和 .xls 格式")
		return
	}

	// 读取文件内容（用于项目持久化 latest.xlsx）
	content, err := io.ReadAll(file)
	if err != nil {
		errorResponse(c, 1002, "读取文件失败")
		return
	}
	if int64(len(content)) != header.Size {
		// 非致命：继续处理
	}

	parser := excel.NewParser()
	if err := parser.LoadFile(bytes.NewReader(content)); err != nil {
		errorResponse(c, 1002, "文件解析失败: "+err.Error())
		return
	}

	sheets, err := parser.GetSheets()
	if err != nil {
		errorResponse(c, 1002, "获取工作表失败")
		return
	}

	fileID := parser.GetFileID()

	// 缓存parser
	h.parsersMu.Lock()
	h.parsers[fileID] = parser
	h.parsersMu.Unlock()

	h.uploadsMu.Lock()
	h.uploads[fileID] = &uploadedFile{
		FileName: header.Filename,
		Bytes:    content,
	}
	h.uploadsMu.Unlock()

	success(c, gin.H{
		"fileId":   fileID,
		"fileName": header.Filename,
		"fileSize": header.Size,
		"sheets":   sheets,
	})
}

// GetColumns 获取列信息
func (h *Handlers) GetColumns(c *gin.Context) {
	fileID := c.Param("fileId")
	sheet := c.Query("sheet")

	h.parsersMu.RLock()
	parser, ok := h.parsers[fileID]
	h.parsersMu.RUnlock()

	if !ok {
		errorResponse(c, 2001, "文件不存在或已过期")
		return
	}

	columns, err := parser.GetColumns(sheet)
	if err != nil {
		errorResponse(c, 2001, "获取列信息失败")
		return
	}

	previewRows, _ := parser.GetPreviewRows(sheet, 5)

	success(c, gin.H{
		"columns":     columns,
		"previewRows": previewRows,
	})
}

// SetMapping 设置字段映射
func (h *Handlers) SetMapping(c *gin.Context) {
	fileID := c.Param("fileId")

	var req struct {
		Sheet   string              `json:"sheet"`
		Mapping *model.FieldMapping `json:"mapping"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	h.parsersMu.RLock()
	parser, ok := h.parsers[fileID]
	h.parsersMu.RUnlock()

	if !ok {
		errorResponse(c, 2001, "文件不存在或已过期")
		return
	}

	parser.SetMapping(req.Mapping)

	h.uploadsMu.Lock()
	if up, ok := h.uploads[fileID]; ok {
		up.Mapping = req.Mapping
	}
	h.uploadsMu.Unlock()

	success(c, gin.H{
		"validRows":   0,
		"invalidRows": 0,
		"warnings":    []string{},
	})
}

// ExecuteImport 执行导入
func (h *Handlers) ExecuteImport(c *gin.Context) {
	fileID := c.Param("fileId")

	var req struct {
		Sheet           string `json:"sheet"`
		GenerateHistory bool   `json:"generateHistory"`
		CurrentMonth    int    `json:"currentMonth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	h.parsersMu.RLock()
	parser, ok := h.parsers[fileID]
	h.parsersMu.RUnlock()

	if !ok {
		errorResponse(c, 2001, "文件不存在或已过期")
		return
	}

	companies, err := parser.Parse(req.Sheet)
	if err != nil {
		errorResponse(c, 3001, "解析失败: "+err.Error())
		return
	}

	// 生成历史数据
	generatedCount := 0
	if req.GenerateHistory {
		generator := excel.NewGenerator()
		generatedCount = generator.BatchGenerateHistory(companies, nil)
	}

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	// 保存到存储
	h.store.SetCompanies(companies)

	// 更新配置
	if req.CurrentMonth > 0 {
		h.store.UpdateConfig(map[string]interface{}{
			"currentMonth": req.CurrentMonth,
		})
	}

	// 计算指标
	indicators := h.engine.Calculate()

	if h.projects != nil {
		cp, _ := h.projects.Current()
		if cp != nil && cp.Project.ProjectID != "" {
			h.uploadsMu.RLock()
			upload := h.uploads[fileID]
			h.uploadsMu.RUnlock()

			if upload != nil && len(upload.Bytes) > 0 {
				_ = h.projects.SaveLatestXlsx(cp.Project.ProjectID, upload.Bytes)
				h.projects.UpdateImportMeta(upload.FileName, req.Sheet, len(companies), generatedCount)
			} else {
				h.projects.UpdateImportMeta("", req.Sheet, len(companies), generatedCount)
			}
			_ = h.projects.SaveNow()
		}
	}

	success(c, gin.H{
		"importedCount":         len(companies),
		"generatedHistoryCount": generatedCount,
		"indicators":            indicators,
	})
}

// ListCompanies 获取企业列表
func (h *Handlers) ListCompanies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	search := c.Query("search")
	industry := c.Query("industry")
	scale := c.Query("scale")
	sortBy := strings.TrimSpace(c.Query("sortBy"))
	sortDir := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sortDir", "asc")))

	companies := h.store.GetAllCompanies()

	// 筛选
	filtered := make([]*model.Company, 0, len(companies))
	for _, c := range companies {
		// 搜索
		if search != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(search)) {
			continue
		}
		// 行业筛选
		if industry != "" && string(c.IndustryType) != industry {
			continue
		}
		// 规模筛选
		if scale != "" {
			scales := strings.Split(scale, ",")
			match := false
			for _, s := range scales {
				if si, err := strconv.Atoi(s); err == nil && c.CompanyScale == si {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, c)
	}

	// 排序（在分页前执行）；默认按名称升序，保证稳定
	if sortBy == "" {
		sortBy = "name"
	}
	desc := sortDir != "asc"
	sort.SliceStable(filtered, func(i, j int) bool {
		a := filtered[i]
		b := filtered[j]
		less := false
		switch sortBy {
		case "name":
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "salesCurrentMonth":
			less = a.SalesCurrentMonth < b.SalesCurrentMonth
		case "retailCurrentMonth":
			less = a.RetailCurrentMonth < b.RetailCurrentMonth
		case "retailLastYearMonth":
			less = a.RetailLastYearMonth < b.RetailLastYearMonth
		case "monthGrowthRate":
			less = a.MonthGrowthRate() < b.MonthGrowthRate()
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			return !less
		}
		return less
	})

	// 分页
	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	items := filtered[start:end]

	// 添加校验信息
	type CompanyWithValidation struct {
		*model.Company
		MonthGrowthRate      float64                `json:"monthGrowthRate"`
		CumulativeGrowthRate float64                `json:"cumulativeGrowthRate"`
		Validation           map[string]interface{} `json:"validation"`
	}

	result := make([]CompanyWithValidation, 0, len(items))
	for _, item := range items {
		errors := item.Validate()
		hasError := false
		for _, e := range errors {
			if e.Severity == "error" {
				hasError = true
				break
			}
		}

		result = append(result, CompanyWithValidation{
			Company:              item,
			MonthGrowthRate:      item.MonthGrowthRate(),
			CumulativeGrowthRate: item.CumulativeGrowthRate(),
			Validation: map[string]interface{}{
				"hasError": hasError,
				"errors":   errors,
			},
		})
	}

	success(c, gin.H{
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"items":    result,
	})
}

// GetCompany 获取单个企业
func (h *Handlers) GetCompany(c *gin.Context) {
	id := c.Param("id")

	company, err := h.store.GetCompany(id)
	if err != nil {
		errorResponse(c, 2001, "企业不存在")
		return
	}

	success(c, company)
}

// UpdateCompany 更新企业数据
func (h *Handlers) UpdateCompany(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name                *string  `json:"name"`
		RetailCurrentMonth  *float64 `json:"retailCurrentMonth"`
		RetailLastYearMonth *float64 `json:"retailLastYearMonth"`
		SalesCurrentMonth   *float64 `json:"salesCurrentMonth"`
		UpdateSales         bool     `json:"updateSales"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	if req.Name == nil && req.RetailCurrentMonth == nil && req.RetailLastYearMonth == nil && req.SalesCurrentMonth == nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	var company *model.Company

	if _, err := h.store.GetCompany(id); err != nil {
		errorResponse(c, 2001, "企业不存在")
		return
	}

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	if req.Name != nil {
		if _, err := h.store.UpdateCompanyName(id, *req.Name); err != nil {
			errorResponse(c, 2001, "企业不存在")
			return
		}
	}

	if req.RetailCurrentMonth != nil {
		if _, err := h.store.UpdateCompanyRetail(id, *req.RetailCurrentMonth); err != nil {
			errorResponse(c, 2001, "企业不存在")
			return
		}
	}

	if req.RetailLastYearMonth != nil {
		if _, err := h.store.UpdateCompanyRetailLastYearMonth(id, *req.RetailLastYearMonth); err != nil {
			errorResponse(c, 2001, "企业不存在")
			return
		}
	}

	if req.SalesCurrentMonth != nil {
		if _, err := h.store.UpdateCompanySales(id, *req.SalesCurrentMonth); err != nil {
			errorResponse(c, 2001, "企业不存在")
			return
		}
	}

	company, err := h.store.GetCompany(id)
	if err != nil {
		errorResponse(c, 2001, "企业不存在")
		return
	}

	// 校验
	errors := company.Validate()
	hasError := false
	for _, e := range errors {
		if e.Severity == "error" {
			hasError = true
			break
		}
	}

	// 重新计算指标
	indicators := h.engine.Calculate()

	if h.projects != nil {
		h.projects.ScheduleSave()
	}

	success(c, gin.H{
		"company": gin.H{
			"id":                      company.ID,
			"name":                    company.Name,
			"salesCurrentMonth":       company.SalesCurrentMonth,
			"retailCurrentMonth":      company.RetailCurrentMonth,
			"retailLastYearMonth":     company.RetailLastYearMonth,
			"retailCurrentCumulative": company.RetailCurrentCumulative,
			"monthGrowthRate":         company.MonthGrowthRate(),
			"validation": gin.H{
				"hasError": hasError,
				"errors":   errors,
			},
		},
		"indicators": indicators,
	})
}

// BatchUpdateCompanies 批量更新企业
func (h *Handlers) BatchUpdateCompanies(c *gin.Context) {
	var req struct {
		Updates []struct {
			ID                 string  `json:"id"`
			RetailCurrentMonth float64 `json:"retailCurrentMonth"`
		} `json:"updates"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	updates := make(map[string]float64)
	for _, u := range req.Updates {
		updates[u.ID] = u.RetailCurrentMonth
	}

	if len(updates) == 0 {
		errorResponse(c, 1001, "参数错误")
		return
	}

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	h.store.BatchUpdateCompanyRetail(updates)

	indicators := h.engine.Calculate()

	if h.projects != nil {
		h.projects.ScheduleSave()
	}

	success(c, gin.H{
		"updatedCount": len(updates),
		"indicators":   indicators,
	})
}

// ResetCompanies 重置企业数据
func (h *Handlers) ResetCompanies(c *gin.Context) {
	var req struct {
		CompanyIds []string `json:"companyIds"`
	}
	c.ShouldBindJSON(&req)

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	h.store.ResetCompanies(req.CompanyIds)

	indicators := h.engine.Calculate()

	if h.projects != nil {
		h.projects.ScheduleSave()
	}

	success(c, gin.H{
		"indicators": indicators,
	})
}

// GetIndicators 获取指标
func (h *Handlers) GetIndicators(c *gin.Context) {
	indicators := h.engine.Calculate()
	success(c, indicators)
}

// AdjustIndicator 调整指标（把“编辑指标”映射为底层数据变更）
func (h *Handlers) AdjustIndicator(c *gin.Context) {
	var req struct {
		Key   string  `json:"key"`
		Value float64 `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		errorResponse(c, 1001, "参数错误")
		return
	}
	if !isSupportedAdjustKey(key) {
		errorResponse(c, 3003, "unsupported key")
		return
	}

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	indicators, err := h.adjuster.Adjust(key, req.Value)
	if err != nil {
		errorResponse(c, 3003, err.Error())
		return
	}

	if h.projects != nil {
		h.projects.ScheduleSave()
	}

	success(c, indicators)
}

func isSupportedAdjustKey(key string) bool {
	key = strings.TrimSpace(key)
	switch {
	case key == "limitAboveMonthValue":
		return true
	case key == "limitAboveMonthRate":
		return true
	case key == "limitAboveCumulativeValue":
		return true
	case key == "limitAboveCumulativeRate":
		return true
	case key == "eatWearUseMonthRate":
		return true
	case key == "microSmallMonthRate":
		return true
	case key == "totalSocialCumulativeValue":
		return true
	case key == "totalSocialCumulativeRate":
		return true
	case strings.HasPrefix(key, "industry."):
		parts := strings.Split(key, ".")
		if len(parts) != 3 {
			return false
		}
		period := parts[2]
		return period == "monthRate" || period == "cumulativeRate"
	default:
		return false
	}
}

// Optimize 执行智能调整
func (h *Handlers) Optimize(c *gin.Context) {
	var req struct {
		TargetIndicator string                     `json:"targetIndicator"`
		TargetValue     float64                    `json:"targetValue"`
		Constraints     *model.OptimizeConstraints `json:"constraints"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	result, err := h.optimizer.Optimize(req.TargetValue, req.Constraints)
	if err != nil {
		errorResponse(c, 3002, err.Error())
		return
	}

	if h.projects != nil {
		h.projects.ScheduleSave()
	}

	success(c, result)
}

// PreviewOptimize 预览智能调整
func (h *Handlers) PreviewOptimize(c *gin.Context) {
	var req struct {
		TargetIndicator string                     `json:"targetIndicator"`
		TargetValue     float64                    `json:"targetValue"`
		Constraints     *model.OptimizeConstraints `json:"constraints"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	result, err := h.optimizer.Preview(req.TargetValue, req.Constraints)
	if err != nil && result == nil {
		errorResponse(c, 3002, err.Error())
		return
	}

	success(c, result)
}

// GetConfig 获取配置
func (h *Handlers) GetConfig(c *gin.Context) {
	config := h.store.GetConfig()
	success(c, config)
}

// UpdateConfig 更新配置
func (h *Handlers) UpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, 1001, "参数错误")
		return
	}

	if h.projects != nil {
		_ = h.projects.SaveUndoSnapshot()
	}

	h.store.UpdateConfig(req)

	// 重新计算指标
	indicators := h.engine.Calculate()

	if h.projects != nil {
		h.projects.ScheduleSave()
	}

	success(c, gin.H{
		"config":     h.store.GetConfig(),
		"indicators": indicators,
	})
}

// Export 导出数据
func (h *Handlers) Export(c *gin.Context) {
	var req struct {
		Format            string `json:"format"`
		IncludeIndicators bool   `json:"includeIndicators"`
		IncludeChanges    bool   `json:"includeChanges"`
	}
	c.ShouldBindJSON(&req)

	companies := h.store.GetAllCompanies()
	var indicators *model.Indicators
	if req.IncludeIndicators {
		indicators = h.engine.Calculate()
	}

	exporter := excel.NewExporter()
	file, err := exporter.Export(companies, indicators, req.IncludeChanges)
	if err != nil {
		errorResponse(c, 3001, "导出失败")
		return
	}

	// 保存临时文件
	exportID := uuid.New().String()
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("northstar_export_%s.xlsx", exportID))
	if err := file.SaveAs(tmpPath); err != nil {
		errorResponse(c, 3001, "保存失败")
		return
	}

	// 缓存路径
	h.exportsMu.Lock()
	h.exports[exportID] = tmpPath
	h.exportsMu.Unlock()

	success(c, gin.H{
		"downloadUrl": fmt.Sprintf("/api/v1/export/download/%s", exportID),
		"expiresAt":   time.Now().Add(time.Hour).Format(time.RFC3339),
	})
}

// Download 下载导出文件
func (h *Handlers) Download(c *gin.Context) {
	exportID := c.Param("exportId")

	h.exportsMu.RLock()
	path, ok := h.exports[exportID]
	h.exportsMu.RUnlock()

	if !ok {
		c.String(http.StatusNotFound, "文件不存在或已过期")
		return
	}

	c.Header("Content-Disposition", "attachment; filename=northstar_export.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.File(path)
}
