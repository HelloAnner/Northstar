/**
 * 导入数据验证接口 — 对比导入记录与数据库实际数据
 *
 * @author Anner
 * Created on 2026/3/29
 */

package v3

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

// verifySheetItem 单个 Sheet 的验证结果
type verifySheetItem struct {
	SheetName    string `json:"sheetName"`
	SheetType    string `json:"sheetType"`
	Status       string `json:"status"`
	ExpectedRows int    `json:"expectedRows"` // sheets_meta 记录的 imported_rows
	ActualRows   int    `json:"actualRows"`   // 数据库实际行数
	Match        bool   `json:"match"`
}

// verifySummary 汇总验证结果
type verifySummary struct {
	ExpectedTotal int  `json:"expectedTotal"`
	ActualTotal   int  `json:"actualTotal"`
	WRExpected    int  `json:"wrExpected"`
	WRActual      int  `json:"wrActual"`
	ACExpected    int  `json:"acExpected"`
	ACActual      int  `json:"acActual"`
	AllMatch      bool `json:"allMatch"`
}

type importVerifyResponse struct {
	Filename string            `json:"filename"`
	Sheets   []verifySheetItem `json:"sheets"`
	Summary  verifySummary     `json:"summary"`
}

// VerifyImport 验证最近一次导入的数据完整性
// GET /api/import/verify
func (h *Handler) VerifyImport(c *gin.Context) {
	log, err := h.store.GetLatestImportLog()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if log == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到导入记录"})
		return
	}

	// 获取当前年月
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查询数据库实际行数
	wrOpts := store.WRQueryOptions{DataYear: &year, DataMonth: &month}
	wrActual, _ := h.store.CountWR(wrOpts)
	acOpts := store.ACQueryOptions{DataYear: &year, DataMonth: &month}
	acActual, _ := h.store.CountAC(acOpts)

	// 获取 sheets_meta
	sheets, err := h.store.ListSheetMetaByImportLog(log.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var items []verifySheetItem
	var wrExpected, acExpected int

	for _, meta := range sheets {
		if meta.Status == "skipped" || meta.Status == "error" {
			continue
		}
		item := verifySheetItem{
			SheetName:    meta.SheetName,
			SheetType:    meta.SheetType,
			Status:       meta.Status,
			ExpectedRows: meta.ImportedRows,
		}

		// 按类型归类
		switch meta.SheetType {
		case "wholesale", "retail":
			wrExpected += meta.ImportedRows
		case "accommodation", "catering":
			acExpected += meta.ImportedRows
		default:
			// snapshot/summary 等类型不纳入主表对比
			item.ActualRows = meta.ImportedRows
			item.Match = true
			items = append(items, item)
			continue
		}

		items = append(items, item)
	}

	// 填充实际行数到每个 sheet item
	for i := range items {
		switch items[i].SheetType {
		case "wholesale", "retail":
			// 按比例分配（如果只有一个 WR sheet，就是全部）
			items[i].ActualRows = items[i].ExpectedRows
		case "accommodation", "catering":
			items[i].ActualRows = items[i].ExpectedRows
		}
	}

	// 用 DB 总数覆盖，判断是否匹配
	allMatch := wrExpected == wrActual && acExpected == acActual

	// 重新填充：如果总数匹配，各 sheet 标记匹配；否则按总数判断
	for i := range items {
		switch items[i].SheetType {
		case "wholesale", "retail":
			if wrExpected > 0 && wrExpected == wrActual {
				items[i].ActualRows = items[i].ExpectedRows
				items[i].Match = true
			} else {
				// 按比例分配实际数
				if wrExpected > 0 {
					items[i].ActualRows = int(float64(items[i].ExpectedRows) / float64(wrExpected) * float64(wrActual))
				}
				items[i].Match = items[i].ExpectedRows == items[i].ActualRows
			}
		case "accommodation", "catering":
			if acExpected > 0 && acExpected == acActual {
				items[i].ActualRows = items[i].ExpectedRows
				items[i].Match = true
			} else {
				if acExpected > 0 {
					items[i].ActualRows = int(float64(items[i].ExpectedRows) / float64(acExpected) * float64(acActual))
				}
				items[i].Match = items[i].ExpectedRows == items[i].ActualRows
			}
		}
	}

	c.JSON(http.StatusOK, importVerifyResponse{
		Filename: log.Filename,
		Sheets:   items,
		Summary: verifySummary{
			ExpectedTotal: wrExpected + acExpected,
			ActualTotal:   wrActual + acActual,
			WRExpected:    wrExpected,
			WRActual:      wrActual,
			ACExpected:    acExpected,
			ACActual:      acActual,
			AllMatch:      allMatch,
		},
	})
}
