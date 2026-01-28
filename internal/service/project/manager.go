package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"northstar/internal/model"
	"northstar/internal/service/calculator"
	"northstar/internal/service/store"
)

const (
	schemaVersion     = 1
	saveDebounceDelay = time.Second
)

// Manager 项目管理器：负责索引维护、切换项目、持久化与自动保存
type Manager struct {
	dataDir string

	store  *store.MemoryStore
	engine *calculator.Engine

	mu         sync.Mutex
	index      ProjectsIndex
	activeID   string
	saveTimer  *time.Timer
	lastImport ImportHistoryItem
	hasImport  bool
}

func NewManager(dataDir string, store *store.MemoryStore, engine *calculator.Engine) (*Manager, error) {
	if err := requireNonEmptyString(dataDir, "dataDir is required"); err != nil {
		return nil, err
	}

	m := &Manager{
		dataDir: dataDir,
		store:   store,
		engine:  engine,
		index: ProjectsIndex{
			SchemaVersion: schemaVersion,
			Items:         []ProjectSummary{},
		},
	}

	if err := m.loadIndex(); err != nil {
		return nil, err
	}
	m.activeID = m.index.LastActiveProjectID
	if m.activeID != "" {
		_ = m.loadProjectState(m.activeID)
	}
	return m, nil
}

func (m *Manager) indexPath() string {
	return filepath.Join(m.dataDir, "projects.json")
}

func (m *Manager) projectDir(projectID string) string {
	return filepath.Join(m.dataDir, projectID)
}

func (m *Manager) metaPath(projectID string) string {
	return filepath.Join(m.projectDir(projectID), "meta.json")
}

func (m *Manager) statePath(projectID string) string {
	return filepath.Join(m.projectDir(projectID), "state.json")
}

func (m *Manager) undoStatePath(projectID string) string {
	return filepath.Join(m.projectDir(projectID), "undo_state.json")
}

func (m *Manager) historyPath(projectID string) string {
	return filepath.Join(m.projectDir(projectID), "import_history.json")
}

func (m *Manager) latestXlsxPath(projectID string) string {
	return filepath.Join(m.projectDir(projectID), "latest.xlsx")
}

func (m *Manager) loadIndex() error {
	path := m.indexPath()
	if !fileExists(path) {
		return writeJSONAtomic(path, m.index)
	}
	var idx ProjectsIndex
	if err := readJSON(path, &idx); err != nil {
		return err
	}
	if idx.SchemaVersion == 0 {
		idx.SchemaVersion = schemaVersion
	}
	m.index = idx
	return nil
}

func (m *Manager) saveIndexLocked() error {
	return writeJSONAtomic(m.indexPath(), m.index)
}

func (m *Manager) ListProjects() ProjectsIndex {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshHasDataLocked()
	return m.index
}

func (m *Manager) Current() (*CurrentProject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeID == "" {
		return &CurrentProject{
			Project: ProjectSummary{},
			HasData: false,
		}, nil
	}

	summary, ok := m.findProjectLocked(m.activeID)
	if !ok {
		return nil, errors.New("current project not found")
	}
	return &CurrentProject{Project: summary, HasData: summary.HasData}, nil
}

func (m *Manager) CreateProject(name string) (ProjectSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := requireNonEmptyString(name, "name is required"); err != nil {
		return ProjectSummary{}, err
	}

	now := time.Now().UTC()
	projectID := fmt.Sprintf("p_%s", uuid.New().String()[:8])
	summary := ProjectSummary{
		ProjectID:    projectID,
		Name:         name,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastOpenedAt: now,
		HasData:      false,
	}

	meta := ProjectMeta{
		SchemaVersion: schemaVersion,
		ProjectID:     projectID,
		Name:          name,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := writeJSONAtomic(m.metaPath(projectID), meta); err != nil {
		return ProjectSummary{}, err
	}
	if err := writeJSONAtomic(m.historyPath(projectID), []ImportHistoryItem{}); err != nil {
		return ProjectSummary{}, err
	}

	m.index.Items = append(m.index.Items, summary)
	m.index.LastActiveProjectID = projectID
	m.activeID = projectID

	m.store.Clear()
	m.store.SetConfig(store.NewMemoryStore().GetConfig())

	if err := m.saveIndexLocked(); err != nil {
		return ProjectSummary{}, err
	}
	return summary, nil
}

func (m *Manager) SelectProject(projectID string) (ProjectSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := requireNonEmptyString(projectID, "projectId is required"); err != nil {
		return ProjectSummary{}, err
	}

	if m.activeID != "" && m.activeID != projectID {
		if err := m.saveNowLocked(); err != nil {
			return ProjectSummary{}, err
		}
	}

	summary, ok := m.findProjectLocked(projectID)
	if !ok {
		return ProjectSummary{}, errors.New("project not found")
	}

	now := time.Now().UTC()
	summary.LastOpenedAt = now
	m.replaceProjectLocked(summary)

	m.index.LastActiveProjectID = projectID
	m.activeID = projectID
	_ = m.loadProjectState(projectID)

	if err := m.saveIndexLocked(); err != nil {
		return ProjectSummary{}, err
	}
	return summary, nil
}

func (m *Manager) GetProjectDetail(projectID string) (*ProjectDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	summary, ok := m.findProjectLocked(projectID)
	if !ok {
		return nil, errors.New("project not found")
	}

	var meta ProjectMeta
	if err := readJSON(m.metaPath(projectID), &meta); err != nil {
		return nil, err
	}

	history := []ImportHistoryItem{}
	if fileExists(m.historyPath(projectID)) {
		_ = readJSON(m.historyPath(projectID), &history)
	}

	return &ProjectDetail{
		Project: summary,
		Meta:    meta,
		History: history,
	}, nil
}

func (m *Manager) DeleteProject(projectID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := requireNonEmptyString(projectID, "projectId is required"); err != nil {
		return err
	}

	if _, ok := m.findProjectLocked(projectID); !ok {
		return errors.New("project not found")
	}

	// 从索引移除
	nextItems := make([]ProjectSummary, 0, len(m.index.Items))
	for _, item := range m.index.Items {
		if item.ProjectID == projectID {
			continue
		}
		nextItems = append(nextItems, item)
	}
	m.index.Items = nextItems

	// 删除目录（尽量在保存索引前执行，避免索引指向不存在目录）
	_ = os.RemoveAll(m.projectDir(projectID))

	// 如果删除的是当前项目，清空 active 并重置 store
	if m.activeID == projectID {
		m.activeID = ""
		m.index.LastActiveProjectID = ""
		if m.index.LastEditedProjectID == projectID {
			m.index.LastEditedProjectID = ""
		}
		m.store.Clear()
		m.store.SetConfig(store.NewMemoryStore().GetConfig())
	} else if m.index.LastActiveProjectID == projectID {
		m.index.LastActiveProjectID = m.activeID
		if m.index.LastEditedProjectID == projectID {
			m.index.LastEditedProjectID = m.activeID
		}
	}

	return m.saveIndexLocked()
}

func (m *Manager) UpdateImportMeta(fileName string, sheet string, importedCount int, generatedCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hasImport = true
	m.lastImport = ImportHistoryItem{
		ImportedAt:     time.Now().UTC(),
		FileName:       fileName,
		Sheet:          sheet,
		ImportedCount:  importedCount,
		GeneratedCount: generatedCount,
	}
}

func (m *Manager) SaveNow() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveNowLocked()
}

func (m *Manager) ScheduleSave() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeID == "" {
		return
	}

	if m.saveTimer != nil {
		m.saveTimer.Stop()
	}
	m.saveTimer = time.AfterFunc(saveDebounceDelay, func() {
		_ = m.SaveNow()
	})
}

func (m *Manager) SaveLatestXlsx(projectID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if projectID == "" {
		return errors.New("projectId is required")
	}
	path := m.latestXlsxPath(projectID)
	return writeBytesAtomic(path, data)
}

// SaveUndoSnapshot 保存“撤销”快照（单步）：在每次数据写入前调用，记录写入前的状态
func (m *Manager) SaveUndoSnapshot() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeID == "" {
		return nil
	}

	state := struct {
		SchemaVersion int              `json:"schemaVersion"`
		ProjectID     string           `json:"projectId"`
		SavedAt       time.Time        `json:"savedAt"`
		Config        *model.Config    `json:"config"`
		Companies     []*model.Company `json:"companies"`
	}{
		SchemaVersion: schemaVersion,
		ProjectID:     m.activeID,
		SavedAt:       time.Now().UTC(),
		Config:        m.store.GetConfig(),
		Companies:     m.store.GetAllCompanies(),
	}

	return writeJSONAtomic(m.undoStatePath(m.activeID), state)
}

// UndoLast 撤销上一次修改：恢复为上一次快照，并清空快照（单步撤销）
func (m *Manager) UndoLast() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeID == "" {
		return errors.New("no active project")
	}

	path := m.undoStatePath(m.activeID)
	if !fileExists(path) {
		return errors.New("no undo snapshot")
	}

	var state struct {
		Config    *model.Config    `json:"config"`
		Companies []*model.Company `json:"companies"`
	}
	if err := readJSON(path, &state); err != nil {
		return err
	}

	if state.Config != nil {
		m.store.SetConfig(state.Config)
	}
	if state.Companies != nil {
		m.store.SetCompanies(state.Companies)
	}

	_ = os.Remove(path)
	return nil
}

func writeBytesAtomic(path string, data []byte) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := osWriteFile(tmp, data); err != nil {
		return err
	}
	return osRename(tmp, path)
}

var (
	osWriteFile = func(path string, data []byte) error { return os.WriteFile(path, data, 0644) }
	osRename    = func(old string, new string) error { return os.Rename(old, new) }
)

func (m *Manager) saveNowLocked() error {
	if m.activeID == "" {
		return nil
	}

	now := time.Now().UTC()

	companies := m.store.GetAllCompanies()
	cfg := m.store.GetConfig()
	indicators := m.engine.Calculate()

	state := struct {
		SchemaVersion int               `json:"schemaVersion"`
		ProjectID     string            `json:"projectId"`
		UpdatedAt     time.Time         `json:"updatedAt"`
		Config        *model.Config     `json:"config"`
		Companies     []*model.Company  `json:"companies"`
		Indicators    *model.Indicators `json:"indicators"`
	}{
		SchemaVersion: schemaVersion,
		ProjectID:     m.activeID,
		UpdatedAt:     now,
		Config:        cfg,
		Companies:     companies,
		Indicators:    indicators,
	}

	if err := writeJSONAtomic(m.statePath(m.activeID), state); err != nil {
		return err
	}

	m.refreshHasDataLocked()
	summary, ok := m.findProjectLocked(m.activeID)
	if ok {
		summary.UpdatedAt = now
		summary.CompanyCount = len(companies)
		if m.hasImport {
			summary.HasData = true
			summary.LastImportAt = m.lastImport.ImportedAt
			summary.LastFileName = m.lastImport.FileName
			summary.LastSheetName = m.lastImport.Sheet
		}
		m.replaceProjectLocked(summary)
	}

	if m.hasImport {
		m.appendImportHistoryLocked(m.activeID, m.lastImport)
		m.hasImport = false
	}

	// 只要触发保存，认为该项目发生过编辑，记录为“上次修改项目”
	m.index.LastEditedProjectID = m.activeID

	return m.saveIndexLocked()
}

func (m *Manager) appendImportHistoryLocked(projectID string, item ImportHistoryItem) {
	history := []ImportHistoryItem{}
	if fileExists(m.historyPath(projectID)) {
		_ = readJSON(m.historyPath(projectID), &history)
	}
	history = append([]ImportHistoryItem{item}, history...)
	if len(history) > 20 {
		history = history[:20]
	}
	_ = writeJSONAtomic(m.historyPath(projectID), history)
}

func (m *Manager) loadProjectState(projectID string) error {
	path := m.statePath(projectID)
	if !fileExists(path) {
		m.store.Clear()
		m.store.SetConfig(store.NewMemoryStore().GetConfig())
		return nil
	}

	var state struct {
		Config    *model.Config    `json:"config"`
		Companies []*model.Company `json:"companies"`
	}
	if err := readJSON(path, &state); err != nil {
		return err
	}

	if state.Config != nil {
		m.store.SetConfig(state.Config)
	}
	if state.Companies != nil {
		m.store.SetCompanies(state.Companies)
	}
	return nil
}

func (m *Manager) refreshHasDataLocked() {
	for i := range m.index.Items {
		id := m.index.Items[i].ProjectID
		m.index.Items[i].HasData = fileExists(m.statePath(id))
	}
	sort.SliceStable(m.index.Items, func(i, j int) bool {
		return m.index.Items[i].LastOpenedAt.After(m.index.Items[j].LastOpenedAt)
	})
}

func (m *Manager) findProjectLocked(projectID string) (ProjectSummary, bool) {
	for _, item := range m.index.Items {
		if item.ProjectID == projectID {
			return item, true
		}
	}
	return ProjectSummary{}, false
}

func (m *Manager) replaceProjectLocked(summary ProjectSummary) {
	for i := range m.index.Items {
		if m.index.Items[i].ProjectID == summary.ProjectID {
			m.index.Items[i] = summary
			return
		}
	}
}
