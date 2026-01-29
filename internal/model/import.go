package model

// ResolveRequest 解析阶段请求参数（手动选择月份 + 人工确认 sheet 类型）
type ResolveRequest struct {
	Month     int                  `json:"month"`
	Overrides map[string]SheetType `json:"overrides"`
}

// ResolveResult 解析阶段产物：选择后的 sheet 角色映射
type ResolveResult struct {
	Month          int                  `json:"month"`
	MainSheets     map[SheetType]string `json:"mainSheets"`
	SnapshotSheets map[SheetType]string `json:"snapshotSheets"`
	UnknownSheets  []string             `json:"unknownSheets"`
	UnusedSheets   []string             `json:"unusedSheets"`
}
