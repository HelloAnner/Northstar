/**
 * LLM 工具调用的数据落库
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package v3

import (
	"fmt"

	"northstar/internal/llm"
)

func (h *Handler) applyLLMCompanyUpdates(updates []llm.CompanyUpdate, year, month int) (fieldExcludes, int, []llmAppliedUpdate, []string, error) {
	excludes := newFieldExcludes()
	warnings := make([]string, 0)
	applied := make([]llmAppliedUpdate, 0)
	updatedCount := 0
	for _, update := range updates {
		kind, id, ok := parseCompanyID(update.ID)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("忽略无效企业 ID: %s", update.ID))
			continue
		}
		if len(update.Patch) == 0 {
			warnings = append(warnings, fmt.Sprintf("忽略空 patch: %s", update.ID))
			continue
		}
		fields, err := h.applyCompanyUpdate(kind, id, update.Patch, year, month)
		if err != nil {
			return excludes, updatedCount, applied, warnings, err
		}
		if len(fields) == 0 {
			continue
		}
		addExclude(&excludes, kind, id, fields)
		applied = append(applied, llmAppliedUpdate{Kind: kind, ID: id, Fields: fields})
		updatedCount++
	}
	return excludes, updatedCount, applied, warnings, nil
}

func (h *Handler) applyCompanyUpdate(kind string, id int64, patch map[string]interface{}, year, month int) ([]string, error) {
	switch kind {
	case "wr":
		return h.applyWRUpdate(id, patch, year, month)
	case "ac":
		return h.applyACUpdate(id, patch, year, month)
	default:
		return nil, fmt.Errorf("未知企业类型: %s", kind)
	}
}

func (h *Handler) applyWRUpdate(id int64, patch map[string]interface{}, year, month int) ([]string, error) {
	existing, err := h.store.GetWRByID(id)
	if err != nil {
		return nil, err
	}
	updates := pickWRUpdates(patch)
	rateUpdates, err := buildWRRateDrivenUpdates(*existing, patch)
	if err != nil {
		return nil, err
	}
	mergeUpdates(updates, rateUpdates)
	if len(updates) == 0 {
		return nil, nil
	}
	if err := h.store.UpdateWR(id, updates); err != nil {
		return nil, err
	}
	if err := recalcDerivedFields(h.store, year, month); err != nil {
		return nil, err
	}
	return updateFieldList(updates), nil
}

func (h *Handler) applyACUpdate(id int64, patch map[string]interface{}, year, month int) ([]string, error) {
	existing, err := h.store.GetACByID(id)
	if err != nil {
		return nil, err
	}
	updates := pickACUpdates(patch)
	rateUpdates, err := buildACRateDrivenUpdates(*existing, patch)
	if err != nil {
		return nil, err
	}
	mergeUpdates(updates, rateUpdates)
	if len(updates) == 0 {
		return nil, nil
	}
	if err := h.store.UpdateAC(id, updates); err != nil {
		return nil, err
	}
	if err := recalcDerivedFields(h.store, year, month); err != nil {
		return nil, err
	}
	return updateFieldList(updates), nil
}

func mergeUpdates(dst map[string]interface{}, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func updateFieldList(updates map[string]interface{}) []string {
	fields := make([]string, 0, len(updates))
	for key := range updates {
		fields = append(fields, key)
	}
	return fields
}

func addExclude(excludes *fieldExcludes, kind string, id int64, fields []string) {
	switch kind {
	case "wr":
		excludes.AddWR(id, fields)
	case "ac":
		excludes.AddAC(id, fields)
	}
}
