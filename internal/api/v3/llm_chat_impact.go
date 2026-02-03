/**
 * LLM 工具影响范围计算
 *
 * @author Anner
 * Created on 2026/2/3
 */

package v3

import (
	"sort"
	"strings"

	"northstar/internal/linkage"
	"northstar/internal/store"
)

type llmAppliedUpdate struct {
	Kind   string
	ID     int64
	Fields []string
}

type llmToolImpact struct {
	ToolPositionCount    int
	ImpactCellCount      int
	ImpactIndicatorCount int
	ImpactCells          []linkage.UICoord
	ImpactIndicators     []string
}

func buildLLMToolImpact(
	st *store.Store,
	year int,
	month int,
	updates []llmAppliedUpdate,
	indicatorTargets map[string]float64,
) (llmToolImpact, error) {
	impact := llmToolImpact{}

	wrRecords, err := st.GetWRByYearMonth(store.WRQueryOptions{DataYear: &year, DataMonth: &month})
	if err != nil {
		return impact, err
	}
	acRecords, err := st.GetACByYearMonth(store.ACQueryOptions{DataYear: &year, DataMonth: &month})
	if err != nil {
		return impact, err
	}

	graph, err := linkage.BuildGraph(linkage.BuildGraphOptions{
		WRRecords: wrRecords,
		ACRecords: acRecords,
	})
	if err != nil {
		return impact, err
	}

	anchorNodes := map[string]linkage.NodeID{}
	for _, update := range updates {
		for _, field := range update.Fields {
			mapped, ok := mapUpdateFieldToNode(update.Kind, field)
			if !ok {
				continue
			}
			node := linkage.BuildNodeID(update.Kind, update.ID, mapped)
			anchorNodes[string(node)] = node
		}
	}

	for id := range indicatorTargets {
		indicatorID := strings.TrimSpace(id)
		if indicatorID == "" {
			continue
		}
		node, err := linkage.ResolveAnchorNode(graph, linkage.Anchor{IndicatorID: indicatorID})
		if err != nil {
			continue
		}
		anchorNodes[string(node)] = node
	}

	impact.ToolPositionCount = len(anchorNodes)
	if len(anchorNodes) == 0 {
		return impact, nil
	}

	impactNodes := map[string]linkage.ImpactNode{}
	for _, node := range anchorNodes {
		for _, item := range linkage.ComputeImpact(graph, node) {
			impactNodes[item.NodeID] = item
		}
	}

	cellMap := map[string]linkage.UICoord{}
	indicatorMap := map[string]struct{}{}
	for _, node := range impactNodes {
		if node.UICoord != nil {
			key := node.UICoord.RowID + "|" + node.UICoord.ColumnKey
			cellMap[key] = *node.UICoord
		}
		if node.IndicatorID != "" {
			indicatorMap[node.IndicatorID] = struct{}{}
		}
	}

	impact.ImpactCells = make([]linkage.UICoord, 0, len(cellMap))
	for _, cell := range cellMap {
		impact.ImpactCells = append(impact.ImpactCells, cell)
	}
	sort.Slice(impact.ImpactCells, func(i, j int) bool {
		if impact.ImpactCells[i].RowID == impact.ImpactCells[j].RowID {
			return impact.ImpactCells[i].ColumnKey < impact.ImpactCells[j].ColumnKey
		}
		return impact.ImpactCells[i].RowID < impact.ImpactCells[j].RowID
	})

	impact.ImpactIndicators = make([]string, 0, len(indicatorMap))
	for id := range indicatorMap {
		impact.ImpactIndicators = append(impact.ImpactIndicators, id)
	}
	sort.Strings(impact.ImpactIndicators)

	impact.ImpactCellCount = len(impact.ImpactCells)
	impact.ImpactIndicatorCount = len(impact.ImpactIndicators)

	return impact, nil
}

func mapUpdateFieldToNode(kind, field string) (string, bool) {
	key := strings.TrimSpace(field)
	if key == "" {
		return "", false
	}
	switch kind {
	case "wr":
		return mapWRFieldToNode(key)
	case "ac":
		return mapACFieldToNode(key)
	default:
		return "", false
	}
}

func mapWRFieldToNode(field string) (string, bool) {
	if v, ok := wrFieldNodeMap[field]; ok {
		return v, true
	}
	return "", false
}

func mapACFieldToNode(field string) (string, bool) {
	if v, ok := acFieldNodeMap[field]; ok {
		return v, true
	}
	return "", false
}

var wrFieldNodeMap = map[string]string{
	"sales_current_month":         "salesCurrentMonth",
	"salesCurrentMonth":           "salesCurrentMonth",
	"sales_last_year_month":       "salesLastYearMonth",
	"salesLastYearMonth":          "salesLastYearMonth",
	"sales_current_cumulative":    "salesCurrentCumulative",
	"salesCurrentCumulative":      "salesCurrentCumulative",
	"sales_last_year_cumulative":  "salesLastYearCumulative",
	"salesLastYearCumulative":     "salesLastYearCumulative",
	"sales_month_rate":            "salesMonthRate",
	"salesMonthRate":              "salesMonthRate",
	"sales_cumulative_rate":       "salesCumulativeRate",
	"salesCumulativeRate":         "salesCumulativeRate",
	"retail_current_month":        "retailCurrentMonth",
	"retailCurrentMonth":          "retailCurrentMonth",
	"retail_last_year_month":      "retailLastYearMonth",
	"retailLastYearMonth":         "retailLastYearMonth",
	"retail_current_cumulative":   "retailCurrentCumulative",
	"retailCurrentCumulative":     "retailCurrentCumulative",
	"retail_last_year_cumulative": "retailLastYearCumulative",
	"retailLastYearCumulative":    "retailLastYearCumulative",
	"retail_month_rate":           "retailMonthRate",
	"retailMonthRate":             "retailMonthRate",
	"retail_cumulative_rate":      "retailCumulativeRate",
	"retailCumulativeRate":        "retailCumulativeRate",
	"is_small_micro":              "flags",
	"isSmallMicro":                "flags",
	"is_eat_wear_use":             "flags",
	"isEatWearUse":                "flags",
}

var acFieldNodeMap = map[string]string{
	"revenue_current_month":        "salesCurrentMonth",
	"revenueCurrentMonth":          "salesCurrentMonth",
	"revenue_last_year_month":      "salesLastYearMonth",
	"revenueLastYearMonth":         "salesLastYearMonth",
	"revenue_current_cumulative":   "salesCurrentCumulative",
	"revenueCurrentCumulative":     "salesCurrentCumulative",
	"revenue_last_year_cumulative": "salesLastYearCumulative",
	"revenueLastYearCumulative":    "salesLastYearCumulative",
	"revenue_month_rate":           "salesMonthRate",
	"revenueMonthRate":             "salesMonthRate",
	"revenue_cumulative_rate":      "salesCumulativeRate",
	"revenueCumulativeRate":        "salesCumulativeRate",
	"room_current_month":           "roomCurrentMonth",
	"roomCurrentMonth":             "roomCurrentMonth",
	"food_current_month":           "foodCurrentMonth",
	"foodCurrentMonth":             "foodCurrentMonth",
	"goods_current_month":          "goodsCurrentMonth",
	"goodsCurrentMonth":            "goodsCurrentMonth",
	"room_current_cumulative":      "roomCurrentCumulative",
	"roomCurrentCumulative":        "roomCurrentCumulative",
	"food_current_cumulative":      "foodCurrentCumulative",
	"foodCurrentCumulative":        "foodCurrentCumulative",
	"goods_current_cumulative":     "goodsCurrentCumulative",
	"goodsCurrentCumulative":       "goodsCurrentCumulative",
	"room_last_year_month":         "roomLastYearMonth",
	"roomLastYearMonth":            "roomLastYearMonth",
	"food_last_year_month":         "foodLastYearMonth",
	"foodLastYearMonth":            "foodLastYearMonth",
	"goods_last_year_month":        "goodsLastYearMonth",
	"goodsLastYearMonth":           "goodsLastYearMonth",
	"room_last_year_cumulative":    "roomLastYearCumulative",
	"roomLastYearCumulative":       "roomLastYearCumulative",
	"food_last_year_cumulative":    "foodLastYearCumulative",
	"foodLastYearCumulative":       "foodLastYearCumulative",
	"goods_last_year_cumulative":   "goodsLastYearCumulative",
	"goodsLastYearCumulative":      "goodsLastYearCumulative",
	"retail_current_month":         "retailCurrentMonth",
	"retailCurrentMonth":           "retailCurrentMonth",
	"retail_last_year_month":       "retailLastYearMonth",
	"retailLastYearMonth":          "retailLastYearMonth",
	"retail_current_cumulative":    "retailCurrentCumulative",
	"retailCurrentCumulative":      "retailCurrentCumulative",
	"retail_last_year_cumulative":  "retailLastYearCumulative",
	"retailLastYearCumulative":     "retailLastYearCumulative",
	"is_small_micro":               "flags",
	"isSmallMicro":                 "flags",
	"is_eat_wear_use":              "flags",
	"isEatWearUse":                 "flags",
}
