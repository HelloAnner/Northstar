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
	n := newNotice(meta, id, target, noticeCodeNoData, "没有可调整的企业数据，无法反推该指标。")
	n.Suggestion = "请先导入企业数据或补齐行业/分类标记。"
	return n
}

func noticeLastYearZero(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeLastYearZero, "上年同期合计为 0，增速无法反推。")
	n.Suggestion = "请补齐上年同期数据，或改调对应的当月值/累计值指标。"
	return n
}

func noticeInvalidTarget(id string, target float64) *OptimizeNotice {
	return &OptimizeNotice{
		IndicatorID: id,
		Target:      target,
		Code:        noticeCodeInvalidTarget,
		Level:       noticeLevel(noticeCodeInvalidTarget),
		Message:     "目标值非法，已跳过调整。",
	}
}

func noticeUnsupported(id string, target float64) *OptimizeNotice {
	return &OptimizeNotice{
		IndicatorID: id,
		Target:      target,
		Code:        noticeCodeUnsupported,
		Level:       noticeLevel(noticeCodeUnsupported),
		Message:     "不支持的指标，已跳过调整。",
	}
}

func noticeTargetSame(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	return newNotice(meta, id, target, noticeCodeTargetSame, "目标值与当前值一致，未触发调整。")
}

func noticeSmallDelta(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeSmallDelta, "目标变化过小，四舍五入后无变化。")
	n.Suggestion = "建议调整幅度 ≥ 1。"
	return n
}

func noticeNoChange(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeNoChange, "调整后未生效，请检查是否存在可调数据。")
	n.Suggestion = "建议刷新后重试，或检查数据范围设置。"
	return n
}

func noticeBelowMin(meta indicatorSnapshot, id string, target float64, minValue float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeBelowMin, "目标低于历史累计下限，系统已按可达下限调整。")
	rounded := roundValue(minValue)
	n.SuggestMin = &rounded
	n.Suggestion = fmt.Sprintf("建议调整到 ≥ %.0f", rounded)
	return n
}

func noticeBelowMinRate(meta indicatorSnapshot, id string, target float64, minRate float64) *OptimizeNotice {
	n := newNotice(meta, id, target, noticeCodeBelowMin, "目标增速低于可达下限，系统已按最小增速调整。")
	rounded := roundValue(minRate)
	n.SuggestMin = &rounded
	n.Suggestion = fmt.Sprintf("建议调整到 ≥ %.0f%%", rounded)
	return n
}

func noticeNotReached(meta indicatorSnapshot, id string, target float64) *OptimizeNotice {
	return newNotice(meta, id, target, noticeCodeNotReached, "目标未完全达成，已按可达范围调整。")
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
