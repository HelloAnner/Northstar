/**
 * 导入预览与原始数据接口
 *
 * @author Anner
 * Created on 2026/2/5
 */

package v3

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"northstar/internal/model"
)

type importPreviewSheet struct {
	SheetName     string                 `json:"sheetName"`
	SheetType     string                 `json:"sheetType"`
	Confidence    float64                `json:"confidence"`
	Status        string                 `json:"status"`
	TotalRows     int                    `json:"totalRows"`
	TotalColumns  int                    `json:"totalColumns"`
	Columns       []string               `json:"columns"`
	ColumnMapping []map[string]any       `json:"columnMapping"`
	Errors        string                 `json:"errors,omitempty"`
	ImportedRows  int                    `json:"importedRows"`
}

type importPreviewResponse struct {
	ImportLog *model.ImportLog     `json:"importLog"`
	Sheets    []importPreviewSheet `json:"sheets"`
}

type importSheetResponse struct {
	SheetName string               `json:"sheetName"`
	MaxRow    int                  `json:"maxRow"`
	MaxCol    int                  `json:"maxCol"`
	Columns   []model.SheetColumn  `json:"columns"`
	Rows      []model.SheetRow     `json:"rows"`
	Cells     []model.SheetCell    `json:"cells"`
	Merges    []model.SheetMerge   `json:"merges"`
}

// GetImportPreview 获取最新导入预览
func (h *Handler) GetImportPreview(c *gin.Context) {
	log, err := h.store.GetLatestImportLog()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if log == nil {
		c.JSON(http.StatusOK, importPreviewResponse{ImportLog: nil, Sheets: []importPreviewSheet{}})
		return
	}

	sheets, err := h.store.ListSheetMetaByImportLog(log.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	respSheets := make([]importPreviewSheet, 0, len(sheets))
	for _, meta := range sheets {
		columns := decodeJSONList(meta.ColumnsJSON)
		mapping := decodeJSONMapList(meta.ColumnMappingJSON)
		respSheets = append(respSheets, importPreviewSheet{
			SheetName:     meta.SheetName,
			SheetType:     meta.SheetType,
			Confidence:    meta.Confidence,
			Status:        meta.Status,
			TotalRows:     meta.TotalRows,
			TotalColumns:  meta.TotalColumns,
			Columns:       columns,
			ColumnMapping: mapping,
			Errors:        meta.ErrorMessage,
			ImportedRows:  meta.ImportedRows,
		})
	}

	c.JSON(http.StatusOK, importPreviewResponse{ImportLog: log, Sheets: respSheets})
}

// GetImportSheet 获取指定 Sheet 的原始预览
func (h *Handler) GetImportSheet(c *gin.Context) {
	sheetName := c.Query("name")
	if sheetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 sheet 名称"})
		return
	}
	log, err := h.store.GetLatestImportLog()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if log == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到导入记录"})
		return
	}

	maxRow := parseInt(c.Query("limitRows"), 200)
	maxCol := parseInt(c.Query("limitCols"), 50)

	columns, err := h.store.GetSheetColumnsByImportLog(log.ID, sheetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.store.GetSheetRowsByImportLog(log.ID, sheetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cells, err := h.store.GetSheetCellsByImportLog(log.ID, sheetName, maxRow, maxCol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	merges, err := h.store.GetSheetMergesByImportLog(log.ID, sheetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, importSheetResponse{
		SheetName: sheetName,
		MaxRow:    maxRow,
		MaxCol:    maxCol,
		Columns:   columns,
		Rows:      rows,
		Cells:     cells,
		Merges:    merges,
	})
}

func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func decodeJSONList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return list
}

func decodeJSONMapList(raw string) []map[string]any {
	if raw == "" {
		return []map[string]any{}
	}
	var list []map[string]any
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []map[string]any{}
	}
	return list
}
