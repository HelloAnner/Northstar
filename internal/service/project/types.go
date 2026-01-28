package project

import "time"

// ProjectSummary 项目概要信息（用于 Project Hub 列表）
type ProjectSummary struct {
	ProjectID     string    `json:"projectId"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	LastOpenedAt  time.Time `json:"lastOpenedAt"`
	HasData       bool      `json:"hasData"`
	CompanyCount  int       `json:"companyCount"`
	LastImportAt  time.Time `json:"lastImportAt"`
	LastFileName  string    `json:"lastFileName"`
	LastSheetName string    `json:"lastSheetName"`
}

// ProjectsIndex 项目索引文件：data/projects.json
type ProjectsIndex struct {
	SchemaVersion       int              `json:"schemaVersion"`
	LastActiveProjectID string           `json:"lastActiveProjectId"`
	LastEditedProjectID string           `json:"lastEditedProjectId"`
	Items               []ProjectSummary `json:"items"`
}

// ProjectMeta 项目目录元信息：data/{projectId}/meta.json
type ProjectMeta struct {
	SchemaVersion int       `json:"schemaVersion"`
	ProjectID     string    `json:"projectId"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ImportHistoryItem 导入记录（用于项目详情页）
type ImportHistoryItem struct {
	ImportedAt     time.Time `json:"importedAt"`
	FileName       string    `json:"fileName"`
	Sheet          string    `json:"sheet"`
	ImportedCount  int       `json:"importedCount"`
	GeneratedCount int       `json:"generatedHistoryCount"`
}

// ProjectDetail 项目详情接口返回（/api/v1/projects/:id）
type ProjectDetail struct {
	Project ProjectSummary      `json:"project"`
	Meta    ProjectMeta         `json:"meta"`
	History []ImportHistoryItem `json:"history"`
}

// CurrentProject 当前选中项目（用于 Topbar 展示）
type CurrentProject struct {
	Project ProjectSummary `json:"project"`
	HasData bool           `json:"hasData"`
}
