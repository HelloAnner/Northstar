package v3

import (
	"fmt"
	"math"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

type OptimizeRequest struct {
	Targets map[string]float64 `json:"targets"`
}

// Optimize 执行智能调整（按目标指标反推并写回企业数据）
// POST /api/optimize
func (h *Handler) Optimize(c *gin.Context) {
	req, ok := bindOptimizeRequest(c)
	if !ok {
		return
	}
	year, month, ok := h.loadCurrentYM(c)
	if !ok {
		return
	}
	h.engine.SetPeriod(year, month)
	resp, err := runOptimize(h.engine, h.store, year, month, req.Targets)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("智能调整失败：%v", err)})
		return
	}
	c.JSON(http.StatusOK, resp)
}

type orderedTarget struct {
	ID    string
	Value float64
}

func applyIndicatorTarget(st *store.Store, year, month int, id string, target float64) error {
	_, err := dagcalc.ApplyIndicatorTarget(st, year, month, id, target, nil, 0)
	return err
}

func orderTargets(targets map[string]float64) []orderedTarget {
	knownOrder := []string{
		"限上社零额_当月值",
		"限上社零额增速_当月",
		"限上社零额_累计值",
		"限上社零额增速_累计",
		"吃穿用增速_当月",
		"小微企业增速_当月",
		"批发业销售额增速_当月",
		"批发业销售额增速_累计",
		"零售业销售额增速_当月",
		"零售业销售额增速_累计",
		"住宿业营业额增速_当月",
		"住宿业营业额增速_累计",
		"餐饮业营业额增速_当月",
		"餐饮业营业额增速_累计",
		"社零总额_累计值",
		"社零总额增速_累计",
	}

	out := make([]orderedTarget, 0, len(targets))
	seen := map[string]bool{}
	for _, id := range knownOrder {
		if v, ok := targets[id]; ok {
			out = append(out, orderedTarget{ID: id, Value: v})
			seen[id] = true
		}
	}
	var rest []string
	for id := range targets {
		if !seen[id] {
			rest = append(rest, id)
		}
	}
	sort.Strings(rest)
	for _, id := range rest {
		out = append(out, orderedTarget{ID: id, Value: targets[id]})
	}
	return out
}

type optimizeResponse struct {
	Year         int                      `json:"year"`
	Month        int                      `json:"month"`
	Groups       []dagcalc.IndicatorGroup `json:"groups"`
	Notices      []OptimizeNotice         `json:"notices,omitempty"`
	AppliedRules []dagcalc.AppliedRule    `json:"appliedRules,omitempty"`
}

func bindOptimizeRequest(c *gin.Context) (OptimizeRequest, bool) {
	var req OptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return OptimizeRequest{}, false
	}
	if len(req.Targets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "targets 不能为空"})
		return OptimizeRequest{}, false
	}
	for k, v := range req.Targets {
		req.Targets[k] = math.Round(v)
	}
	return req, true
}

func (h *Handler) loadCurrentYM(c *gin.Context) (int, int, bool) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前年月失败"})
		return 0, 0, false
	}
	return year, month, true
}

func runOptimize(eng *dagcalc.Engine, st *store.Store, year, month int, targets map[string]float64) (optimizeResponse, error) {
	beforeGroups, err := dagcalc.CalculateIndicators(st, year, month)
	if err != nil {
		return optimizeResponse{}, err
	}
	beforeSnap := buildIndicatorSnapshotMap(beforeGroups)
	ordered := orderTargets(targets)
	noticeMap := map[string]*OptimizeNotice{}
	executableTargets, err := collectExecutableTargets(st, year, month, ordered, beforeSnap, noticeMap)
	if err != nil {
		return optimizeResponse{}, err
	}

	groups := beforeGroups
	appliedRules := []dagcalc.AppliedRule{}
	if len(executableTargets) > 0 {
		result, err := eng.Optimize(executableTargets)
		if err != nil {
			return optimizeResponse{}, err
		}
		groups = result.Groups
		appliedRules = result.AppliedRules
	}

	roundIndicatorGroupsInPlace(groups)
	afterSnap := buildIndicatorSnapshotMap(groups)
	notices, err := finalizeNotices(st, year, month, ordered, beforeSnap, afterSnap, appliedRules, noticeMap)
	if err != nil {
		return optimizeResponse{}, err
	}
	return optimizeResponse{
		Year:         year,
		Month:        month,
		Groups:       groups,
		Notices:      notices,
		AppliedRules: appliedRules,
	}, nil
}

func collectExecutableTargets(
	st *store.Store,
	year, month int,
	ordered []orderedTarget,
	before map[string]indicatorSnapshot,
	noticeMap map[string]*OptimizeNotice,
) (map[string]float64, error) {
	targets := make(map[string]float64, len(ordered))
	for _, item := range ordered {
		meta := before[item.ID]
		notice, err := precheckIndicatorTarget(st, year, month, item.ID, item.Value, meta)
		if err != nil {
			return nil, err
		}
		if notice != nil {
			noticeMap[item.ID] = notice
			continue
		}
		targets[item.ID] = item.Value
	}
	return targets, nil
}

func finalizeNotices(
	st *store.Store,
	year, month int,
	ordered []orderedTarget,
	before map[string]indicatorSnapshot,
	after map[string]indicatorSnapshot,
	appliedRules []dagcalc.AppliedRule,
	noticeMap map[string]*OptimizeNotice,
) ([]OptimizeNotice, error) {
	for _, item := range ordered {
		beforeSnap, afterSnap, ok := snapshotForID(before, after, item.ID)
		if !ok {
			continue
		}
		target := effectiveTargetForNotice(appliedRules, item.ID, item.Value)
		if notice := noticeMap[item.ID]; notice != nil {
			updateNoticeValues(notice, beforeSnap, afterSnap, target)
			continue
		}
		notice, err := buildPostApplyNotice(st, year, month, item.ID, target, beforeSnap, afterSnap)
		if err != nil {
			return nil, err
		}
		if notice != nil {
			noticeMap[item.ID] = notice
		}
	}
	return orderedNotices(ordered, noticeMap), nil
}

func orderedNotices(ordered []orderedTarget, noticeMap map[string]*OptimizeNotice) []OptimizeNotice {
	out := make([]OptimizeNotice, 0, len(noticeMap))
	for _, item := range ordered {
		if notice := noticeMap[item.ID]; notice != nil {
			out = append(out, *notice)
		}
	}
	return out
}

func effectiveTargetForNotice(appliedRules []dagcalc.AppliedRule, indicatorID string, target float64) float64 {
	next := target
	for _, item := range appliedRules {
		if item.Type != "clamp_target" {
			continue
		}
		if item.IndicatorID != indicatorID {
			continue
		}
		next = item.AfterValue
	}
	return next
}
