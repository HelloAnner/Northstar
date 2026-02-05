/**
 * Sheet 原始数据与汇总模型
 *
 * @author Anner
 * Created on 2026/2/5
 */

package model

import "time"

// SheetColumn Sheet 列结构
type SheetColumn struct {
    ID               int64     `json:"id"`
    SheetName        string    `json:"sheetName"`
    ColIdx           int       `json:"colIdx"`
    HeaderText       string    `json:"headerText"`
    NormalizedHeader string    `json:"normalizedHeader"`
    ColWidth         float64   `json:"colWidth"`
    ImportLogID      *int64    `json:"importLogId,omitempty"`
    SourceFile       string    `json:"sourceFile"`
    CreatedAt        time.Time `json:"createdAt"`
}

// SheetRow Sheet 行结构
type SheetRow struct {
    ID          int64     `json:"id"`
    SheetName   string    `json:"sheetName"`
    RowIdx      int       `json:"rowIdx"`
    RowHeight   float64   `json:"rowHeight"`
    ImportLogID *int64    `json:"importLogId,omitempty"`
    SourceFile  string    `json:"sourceFile"`
    CreatedAt   time.Time `json:"createdAt"`
}

// SheetCell Sheet 单元格原始数据
type SheetCell struct {
    ID         int64     `json:"id"`
    SheetName  string    `json:"sheetName"`
    RowIdx     int       `json:"rowIdx"`
    ColIdx     int       `json:"colIdx"`
    A1         string    `json:"a1"`
    CellType   string    `json:"cellType"`
    RawValue   string    `json:"rawValue"`
    Formula    string    `json:"formula"`
    CalcValue  string    `json:"calcValue"`
    NumFormat  string    `json:"numFormat"`
    StyleID    int       `json:"styleId"`
    IsMerged   int       `json:"isMerged"`
    MergeRange string    `json:"mergeRange"`
    ImportLogID *int64   `json:"importLogId,omitempty"`
    SourceFile string    `json:"sourceFile"`
    CreatedAt  time.Time `json:"createdAt"`
}

// SheetMerge Sheet 合并单元格
type SheetMerge struct {
    ID         int64     `json:"id"`
    SheetName  string    `json:"sheetName"`
    MergeRange string    `json:"mergeRange"`
    StartRow   int       `json:"startRow"`
    StartCol   int       `json:"startCol"`
    EndRow     int       `json:"endRow"`
    EndCol     int       `json:"endCol"`
    ImportLogID *int64   `json:"importLogId,omitempty"`
    SourceFile string    `json:"sourceFile"`
    CreatedAt  time.Time `json:"createdAt"`
}

// SummaryLimitAboveRetail 限上零售额汇总
type SummaryLimitAboveRetail struct {
    ID           int64      `json:"id"`
    DataYear     int        `json:"dataYear"`
    DataMonth    int        `json:"dataMonth"`
    RowKey       string     `json:"rowKey"`
    RowNo        int        `json:"rowNo"`
    ValueCurrent *float64   `json:"valueCurrent"`
    ValueLast    *float64   `json:"valueLast"`
    Rate         *float64   `json:"rate"`
    SourceSheet  string     `json:"sourceSheet"`
    SourceCell   string     `json:"sourceCell"`
    ImportLogID  *int64     `json:"importLogId,omitempty"`
    SourceFile   string     `json:"sourceFile"`
    CreatedAt    time.Time  `json:"createdAt"`
}

// SummaryMicroSmall 小微汇总
type SummaryMicroSmall struct {
    ID           int64      `json:"id"`
    DataYear     int        `json:"dataYear"`
    DataMonth    int        `json:"dataMonth"`
    RowKey       string     `json:"rowKey"`
    RowNo        int        `json:"rowNo"`
    ValueCurrent *float64   `json:"valueCurrent"`
    ValueLast    *float64   `json:"valueLast"`
    Rate         *float64   `json:"rate"`
    SourceSheet  string     `json:"sourceSheet"`
    SourceCell   string     `json:"sourceCell"`
    ImportLogID  *int64     `json:"importLogId,omitempty"`
    SourceFile   string     `json:"sourceFile"`
    CreatedAt    time.Time  `json:"createdAt"`
}

// SummaryEatWearUse 吃穿用汇总
type SummaryEatWearUse struct {
    ID           int64      `json:"id"`
    DataYear     int        `json:"dataYear"`
    DataMonth    int        `json:"dataMonth"`
    RowKey       string     `json:"rowKey"`
    RowNo        int        `json:"rowNo"`
    ValueCurrent *float64   `json:"valueCurrent"`
    ValueLast    *float64   `json:"valueLast"`
    Rate         *float64   `json:"rate"`
    SourceSheet  string     `json:"sourceSheet"`
    SourceCell   string     `json:"sourceCell"`
    ImportLogID  *int64     `json:"importLogId,omitempty"`
    SourceFile   string     `json:"sourceFile"`
    CreatedAt    time.Time  `json:"createdAt"`
}

