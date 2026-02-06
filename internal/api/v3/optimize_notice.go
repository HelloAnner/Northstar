/**
 * 智能调整提示信息构造
 *
 * @author Anner
 * Created on 2026/2/4
 */

package v3

import (
	"fmt"
	"math"

	"northstar/internal/dagcalc"
)

const (
	noticeCodeInvalidTarget = "invalid_target"
	noticeCodeUnsupported   = "unsupported_indicator"
	noticeCodeNoData        = "no_data"
	noticeCodeLastYearZero  = "last_year_zero"
	noticeCodeBelowMin      = "below_min"
	noticeCodeTargetSame    = "target_same"
	noticeCodeSmallDelta    = "small_delta"
	noticeCodeNoChange      = "no_change"
	noticeCodeNotReached    = "not_reached"
)

const (
	noticeLevelInfo  = "info"
	noticeLevelWarn  = "warn"
	noticeLevelError = "error"
)

type OptimizeNotice struct {
	IndicatorID   string   `json:"indicatorId"`
	IndicatorName string   `json:"indicatorName"`
	Target        float64  `json:"target"`
	Before        float64  `json:"before"`
	After         float64  `json:"after"`
	Code          string   `json:"code"`
	Level         string   `json:"level"`
	Message       string   `json:"message"`
	Suggestion    string   `json:"suggestion,omitempty"`
	SuggestMin    *float64 `json:"suggestMin,omitempty"`
}

type indicatorSnapshot struct {
	Value float64
	Name  string
	Unit  string
}

func buildIndicatorSnapshotMap(groups []dagcalc.IndicatorGroup) map[string]indicatorSnapshot {
	out := make(map[string]indicatorSnapshot, 24)
	for _, group := range groups {
		for _, indicator := range group.Indicators {
			out[indicator.ID] = indicatorSnapshot{
				Value: indicator.Value,
				Name:  indicator.Name,
				Unit:  indicator.Unit,
			}
		}
	}
	return out
}

func roundValue(v float64) float64 {
	return math.Round(v)
}

func noticeLevel(code string) string {
	switch code {
	case noticeCodeInvalidTarget, noticeCodeUnsupported, noticeCodeNoData, noticeCodeLastYearZero:
		return noticeLevelError
	case noticeCodeTargetSame:
		return noticeLevelInfo
	default:
		return noticeLevelWarn
	}
}

func newNotice(meta indicatorSnapshot, id string, target float64, code string, message string) *OptimizeNotice {
	return &OptimizeNotice{
		IndicatorID:   id,
		IndicatorName: meta.Name,
		Target:        roundValue(target),
		Before:        roundValue(meta.Value),
		After:         roundValue(meta.Value),
		Code:          code,
		Level:         noticeLevel(code),
		Message:       message,
	}
}

func updateNoticeValues(n *OptimizeNotice, before indicatorSnapshot, after indicatorSnapshot, target float64) {
	if n == nil {
		return
	}
	if n.IndicatorName == "" {
		n.IndicatorName = after.Name
	}
	n.Target = roundValue(target)
	n.Before = roundValue(before.Value)
	n.After = roundValue(after.Value)
}

func noticeNoData(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeNoData,
		"根据可调整数据规则无法调整该指标数据，建议先导入企业数据或补齐行业/分类标记。")
	n.Suggestion = "建议先导入企业数据或补齐行业/分类标记。"
	return n
}

func noticeLastYearZero(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeLastYearZero,
		"根据同比基数规则无法调整该指标数据，建议补齐上年同期数据或改调对应的当月值/累计值指标。")
	n.Suggestion = "建议补齐上年同期数据，或改调对应的当月值/累计值指标。"
	return n
}

func noticeInvalidTarget(id string, target float64) *OptimizeNotice {
	return &OptimizeNotice{
		IndicatorID: id,
		Target:      target,
		Code:        noticeCodeInvalidTarget,
		Level:       noticeLevel(noticeCodeInvalidTarget),
		Message:     "根据目标值合法性规则无法调整该指标数据，建议输入有效数字后重试。",
		Suggestion:  "建议输入有效数字后重试。",
	}
}

func noticeUnsupported(id string, target float64) *OptimizeNotice {
	return &OptimizeNotice{
		IndicatorID: id,
		Target:      target,
		Code:        noticeCodeUnsupported,
		Level:       noticeLevel(noticeCodeUnsupported),
		Message:     "根据指标支持范围规则无法调整该指标数据，建议选择当前版本支持的指标。",
		Suggestion:  "建议选择当前版本支持的指标。",
	}
}

func noticeTargetSame(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeTargetSame,
		"根据目标差异规则无法调整该指标数据，建议输入与当前值不同的目标。")
	n.Suggestion = "建议输入与当前值不同的目标。"
	return n
}

func noticeSmallDelta(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeSmallDelta,
		"根据整数取整规则无法调整该指标数据，建议调整幅度至少为 1。")
	n.Suggestion = "建议调整幅度至少为 1。"
	return n
}

func noticeNoChange(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeNoChange,
		"根据可调范围规则无法调整该指标数据，建议刷新后重试或检查数据范围设置。")
	n.Suggestion = "建议刷新后重试，或检查数据范围设置。"
	return n
}

func noticeBelowMin(meta indicatorSnapshot, id string, target float64, minValue float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeBelowMin,
		"根据历史累计下限规则无法调整该指标数据，建议提高目标到可达下限以上。")
	rounded := roundValue(minValue)
	n.SuggestMin = &rounded
	n.Suggestion = fmt.Sprintf("建议调整到 ≥ %.0f", rounded)
	return n
}

func noticeBelowMinRate(meta indicatorSnapshot, id string, target float64, minRate float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeBelowMin,
		"根据增速下限规则无法调整该指标数据，建议提高目标增速到可达下限以上。")
	rounded := roundValue(minRate)
	n.SuggestMin = &rounded
	n.Suggestion = fmt.Sprintf("建议调整到 ≥ %.0f%%", rounded)
	return n
}

func noticeNotReached(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeNotReached,
		"根据当前可达范围规则无法调整该指标数据，建议分步调整或放宽约束后重试。")
	n.Suggestion = "建议分步调整或放宽约束后重试。"
	return n
}

func snapshotForID(before map[string]indicatorSnapshot, after map[string]indicatorSnapshot, id string) (indicatorSnapshot, indicatorSnapshot, bool) {
	beforeSnap, okBefore := before[id]
	afterSnap, okAfter := after[id]
	if !okBefore && okAfter {
		beforeSnap = afterSnap
	}
	if !okAfter && okBefore {
		afterSnap = beforeSnap
	}
	return beforeSnap, afterSnap, okBefore || okAfter
}
