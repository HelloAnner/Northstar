/**
 * 联动 DAG 构图（统一入口）
 *
 * @author Anner
 * Created on 2026/2/4
 */

package dagcalc

import (
	"fmt"

	"northstar/internal/model"
)

const (
	nodeKindWR        = "wr"
	nodeKindAC        = "ac"
	nodeKindIndicator = "indicator"
	nodeKindSummary   = "summary"
)

var uiColumnKeys = []string{
	"companyScale",
	"flags",
	"salesPrevMonth",
	"salesCurrentMonth",
	"salesLastYearMonth",
	"salesYoYDiff",
	"salesMoMDiff",
	"salesMoMRate",
	"salesMonthRate",
	"salesCurrentCumulative",
	"salesLastYearCumulative",
	"salesCumulativeYoYDiff",
	"salesCumulativeRate",
	"retailPrevMonth",
	"retailCurrentMonth",
	"retailLastYearMonth",
	"retailYoYDiff",
	"retailMoMDiff",
	"retailMoMRate",
	"retailMonthRate",
	"retailCurrentCumulative",
	"retailLastYearCumulative",
	"retailCumulativeYoYDiff",
	"retailCumulativeRate",
	"retailRatio",
	"sourceSheet",
}

var indicatorIDs = []string{
	"limitAbove_month_value",
	"limitAbove_month_rate",
	"limitAbove_cumulative_value",
	"limitAbove_cumulative_rate",
	"eatWearUse_month_rate",
	"microSmall_month_rate",
	"wholesale_month_rate",
	"wholesale_cumulative_rate",
	"retail_month_rate",
	"retail_cumulative_rate",
	"accommodation_month_rate",
	"accommodation_cumulative_rate",
	"catering_month_rate",
	"catering_cumulative_rate",
	"totalSocial_cumulative_value",
	"totalSocial_cumulative_rate",
}

// BuildLinkageGraph 构建联动 DAG（统一入口）
func BuildLinkageGraph(index *TemplateIndex, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) (*Graph, error) {
	graph := NewGraph()
	attachIndicatorNodes(graph)

	if err := attachCompanyCoords(graph, index, wrRecords, acRecords); err != nil {
		return nil, err
	}
	attachCompanyEdges(graph, wrRecords, acRecords)
	attachAggregateEdges(graph, wrRecords, acRecords)
	attachSummaryEdges(graph)
	addIndicatorEdges(graph)
	attachIndicatorCoords(graph, index)
	attachAggregateCoords(graph, index)
	attachReverseEdges(graph, wrRecords, acRecords)
	buildIndexes(graph)

	return graph, nil
}

func indicatorNode(id string) NodeID {
	return NodeID(nodeKindIndicator + ":" + id)
}

func summaryNode(kind string, field string) NodeID {
	return NodeID(nodeKindSummary + ":" + kind + ":" + field)
}

func industryNode(industry string, field string) NodeID {
	return NodeID("industry:" + industry + ":" + field)
}

func aggregateNode(field string) NodeID {
	return NodeID("aggregate:" + field)
}

func attachSummaryEdges(g *Graph) {
	g.AddEdge(summaryNode("micro_small", "rate"), indicatorNode("microSmall_month_rate"))
	g.AddEdge(summaryNode("eat_wear_use", "rate"), indicatorNode("eatWearUse_month_rate"))
	g.AddEdge(summaryNode("limit_above", "rate"), indicatorNode("limitAbove_month_rate"))
}

func addWRFieldEdges(g *Graph, r *model.WholesaleRetail) {
	n := func(field string) NodeID {
		return BuildNodeID(nodeKindWR, r.ID, field)
	}

	addSalesEdges(g, n)
	addRetailEdges(g, n)

	g.AddEdge(n("salesCurrentMonth"), n("salesCurrentCumulative"))
	g.AddEdge(n("salesLastYearMonth"), n("salesLastYearCumulative"))
	g.AddEdge(n("retailCurrentMonth"), n("retailCurrentCumulative"))
	g.AddEdge(n("retailLastYearMonth"), n("retailLastYearCumulative"))
}

func addACFieldEdges(g *Graph, r *model.AccommodationCatering) {
	n := func(field string) NodeID {
		return BuildNodeID(nodeKindAC, r.ID, field)
	}

	addSalesEdges(g, n)
	addRetailEdges(g, n)

	g.AddEdge(n("foodCurrentMonth"), n("retailCurrentMonth"))
	g.AddEdge(n("goodsCurrentMonth"), n("retailCurrentMonth"))
	g.AddEdge(n("foodLastYearMonth"), n("retailLastYearMonth"))
	g.AddEdge(n("goodsLastYearMonth"), n("retailLastYearMonth"))
	g.AddEdge(n("foodCurrentCumulative"), n("retailCurrentCumulative"))
	g.AddEdge(n("goodsCurrentCumulative"), n("retailCurrentCumulative"))
	g.AddEdge(n("foodLastYearCumulative"), n("retailLastYearCumulative"))
	g.AddEdge(n("goodsLastYearCumulative"), n("retailLastYearCumulative"))

	g.AddEdge(n("salesCurrentMonth"), n("salesCurrentCumulative"))
	g.AddEdge(n("salesLastYearMonth"), n("salesLastYearCumulative"))
}

func addSalesEdges(g *Graph, n func(string) NodeID) {
	g.AddEdge(n("salesCurrentMonth"), n("salesMonthRate"))
	g.AddEdge(n("salesLastYearMonth"), n("salesMonthRate"))
	g.AddEdge(n("salesCurrentMonth"), n("salesYoYDiff"))
	g.AddEdge(n("salesLastYearMonth"), n("salesYoYDiff"))
	g.AddEdge(n("salesCurrentMonth"), n("salesMoMDiff"))
	g.AddEdge(n("salesPrevMonth"), n("salesMoMDiff"))
	g.AddEdge(n("salesCurrentMonth"), n("salesMoMRate"))
	g.AddEdge(n("salesPrevMonth"), n("salesMoMRate"))
	g.AddEdge(n("salesCurrentCumulative"), n("salesCumulativeRate"))
	g.AddEdge(n("salesLastYearCumulative"), n("salesCumulativeRate"))
	g.AddEdge(n("salesCurrentCumulative"), n("salesCumulativeYoYDiff"))
	g.AddEdge(n("salesLastYearCumulative"), n("salesCumulativeYoYDiff"))
}

func addRetailEdges(g *Graph, n func(string) NodeID) {
	g.AddEdge(n("retailCurrentMonth"), n("retailMonthRate"))
	g.AddEdge(n("retailLastYearMonth"), n("retailMonthRate"))
	g.AddEdge(n("retailCurrentMonth"), n("retailYoYDiff"))
	g.AddEdge(n("retailLastYearMonth"), n("retailYoYDiff"))
	g.AddEdge(n("retailCurrentMonth"), n("retailMoMDiff"))
	g.AddEdge(n("retailPrevMonth"), n("retailMoMDiff"))
	g.AddEdge(n("retailCurrentMonth"), n("retailMoMRate"))
	g.AddEdge(n("retailPrevMonth"), n("retailMoMRate"))
	g.AddEdge(n("retailCurrentCumulative"), n("retailCumulativeRate"))
	g.AddEdge(n("retailLastYearCumulative"), n("retailCumulativeRate"))
	g.AddEdge(n("retailCurrentCumulative"), n("retailCumulativeYoYDiff"))
	g.AddEdge(n("retailLastYearCumulative"), n("retailCumulativeYoYDiff"))
	g.AddEdge(n("retailCurrentMonth"), n("retailRatio"))
	g.AddEdge(n("salesCurrentMonth"), n("retailRatio"))
}

func addIndicatorEdges(g *Graph) {
	g.AddEdge(aggregateNode("limitAboveRetailCurSum"), indicatorNode("limitAbove_month_value"))
	g.AddEdge(aggregateNode("limitAboveRetailCurSum"), indicatorNode("limitAbove_month_rate"))
	g.AddEdge(aggregateNode("limitAboveRetailLastSum"), indicatorNode("limitAbove_month_rate"))

	g.AddEdge(aggregateNode("limitAboveRetailCurCumSum"), indicatorNode("limitAbove_cumulative_value"))
	g.AddEdge(aggregateNode("limitAboveRetailCurCumSum"), indicatorNode("limitAbove_cumulative_rate"))
	g.AddEdge(aggregateNode("limitAboveRetailLastCumSum"), indicatorNode("limitAbove_cumulative_rate"))

	g.AddEdge(aggregateNode("eatWearUseRetailCurSum"), indicatorNode("eatWearUse_month_rate"))
	g.AddEdge(aggregateNode("eatWearUseRetailLastSum"), indicatorNode("eatWearUse_month_rate"))

	g.AddEdge(aggregateNode("microSmallRetailCurSum"), indicatorNode("microSmall_month_rate"))
	g.AddEdge(aggregateNode("microSmallRetailLastSum"), indicatorNode("microSmall_month_rate"))

	g.AddEdge(industryNode("wholesale", "salesCurSum"), indicatorNode("wholesale_month_rate"))
	g.AddEdge(industryNode("wholesale", "salesLastSum"), indicatorNode("wholesale_month_rate"))
	g.AddEdge(industryNode("wholesale", "salesCurCumSum"), indicatorNode("wholesale_cumulative_rate"))
	g.AddEdge(industryNode("wholesale", "salesLastCumSum"), indicatorNode("wholesale_cumulative_rate"))

	g.AddEdge(industryNode("retail", "salesCurSum"), indicatorNode("retail_month_rate"))
	g.AddEdge(industryNode("retail", "salesLastSum"), indicatorNode("retail_month_rate"))
	g.AddEdge(industryNode("retail", "salesCurCumSum"), indicatorNode("retail_cumulative_rate"))
	g.AddEdge(industryNode("retail", "salesLastCumSum"), indicatorNode("retail_cumulative_rate"))

	g.AddEdge(industryNode("accommodation", "salesCurSum"), indicatorNode("accommodation_month_rate"))
	g.AddEdge(industryNode("accommodation", "salesLastSum"), indicatorNode("accommodation_month_rate"))
	g.AddEdge(industryNode("accommodation", "salesCurCumSum"), indicatorNode("accommodation_cumulative_rate"))
	g.AddEdge(industryNode("accommodation", "salesLastCumSum"), indicatorNode("accommodation_cumulative_rate"))

	g.AddEdge(industryNode("catering", "salesCurSum"), indicatorNode("catering_month_rate"))
	g.AddEdge(industryNode("catering", "salesLastSum"), indicatorNode("catering_month_rate"))
	g.AddEdge(industryNode("catering", "salesCurCumSum"), indicatorNode("catering_cumulative_rate"))
	g.AddEdge(industryNode("catering", "salesLastCumSum"), indicatorNode("catering_cumulative_rate"))

	g.AddEdge(indicatorNode("limitAbove_cumulative_value"), indicatorNode("totalSocial_cumulative_value"))
	g.AddEdge(indicatorNode("microSmall_month_rate"), indicatorNode("totalSocial_cumulative_value"))
	g.AddEdge(indicatorNode("totalSocial_cumulative_value"), indicatorNode("totalSocial_cumulative_rate"))
	g.AddEdge(aggregateNode("limitAboveRetailLastCumSum"), indicatorNode("totalSocial_cumulative_rate"))
}

func attachIndicatorNodes(g *Graph) {
	for _, id := range indicatorIDs {
		node := indicatorNode(id)
		g.AddIndicatorID(node, id)
	}
}

func attachCompanyCoords(
	g *Graph,
	index *TemplateIndex,
	wrRecords []*model.WholesaleRetail,
	acRecords []*model.AccommodationCatering,
) error {
	if err := attachWRCoords(g, index, wrRecords); err != nil {
		return err
	}
	if err := attachACCoords(g, index, acRecords); err != nil {
		return err
	}
	return nil
}

func attachWRCoords(g *Graph, index *TemplateIndex, records []*model.WholesaleRetail) error {
	if len(records) == 0 {
		return nil
	}
	allRows, err := buildWRRowMap(index, "批零总表", records)
	if err != nil {
		return err
	}
	whRows, err := buildWRRowMap(index, "批发", filterWRIndustry(records, "wholesale"))
	if err != nil {
		return err
	}
	reRows, err := buildWRRowMap(index, "零售", filterWRIndustry(records, "retail"))
	if err != nil {
		return err
	}

	for _, r := range records {
		rowID := BuildRowID(nodeKindWR, r.ID)
		for _, key := range uiColumnKeys {
			g.AddUICoord(BuildNodeID(nodeKindWR, r.ID, key), UICoord{RowID: rowID, ColumnKey: key})
		}
		appendWRExcelCoords(g, r.ID, allRows[r.ID], "批零总表")
		if r.IndustryType == "wholesale" {
			appendWRExcelCoords(g, r.ID, whRows[r.ID], "批发")
		}
		if r.IndustryType == "retail" {
			appendWRExcelCoords(g, r.ID, reRows[r.ID], "零售")
		}
	}
	return nil
}

func attachACCoords(g *Graph, index *TemplateIndex, records []*model.AccommodationCatering) error {
	if len(records) == 0 {
		return nil
	}
	allRows, err := buildACRowMap(index, "住餐总表", records)
	if err != nil {
		return err
	}
	accRows, err := buildACRowMap(index, "住宿", filterACIndustry(records, "accommodation"))
	if err != nil {
		return err
	}
	catRows, err := buildACRowMap(index, "餐饮", filterACIndustry(records, "catering"))
	if err != nil {
		return err
	}

	for _, r := range records {
		rowID := BuildRowID(nodeKindAC, r.ID)
		for _, key := range uiColumnKeys {
			g.AddUICoord(BuildNodeID(nodeKindAC, r.ID, key), UICoord{RowID: rowID, ColumnKey: key})
		}
		appendACExcelCoords(g, r.ID, allRows[r.ID], "住餐总表")
		if r.IndustryType == "accommodation" {
			appendACExcelCoords(g, r.ID, accRows[r.ID], "住宿")
		}
		if r.IndustryType == "catering" {
			appendACExcelCoords(g, r.ID, catRows[r.ID], "餐饮")
		}
	}
	return nil
}

var wrExcelColumns = map[string]string{
	"salesCurrentMonth":       "D",
	"salesLastYearMonth":      "E",
	"salesMonthRate":          "F",
	"salesCurrentCumulative":  "G",
	"salesLastYearCumulative": "H",
	"salesCumulativeRate":     "I",
	"retailCurrentMonth":      "J",
	"retailLastYearMonth":     "K",
	"retailMonthRate":         "L",
	"retailCurrentCumulative": "M",
	"retailLastYearCumulative": "N",
	"retailCumulativeRate":    "O",
}

var acExcelColumns = map[string]string{
	"salesCurrentMonth":       "D",
	"salesLastYearMonth":      "E",
	"salesMonthRate":          "F",
	"salesCurrentCumulative":  "G",
	"salesLastYearCumulative": "H",
	"salesCumulativeRate":     "I",
	"roomCurrentMonth":        "J",
	"roomLastYearMonth":       "K",
	"roomCurrentCumulative":   "L",
	"roomLastYearCumulative":  "M",
	"foodCurrentMonth":        "N",
	"foodLastYearMonth":       "O",
	"foodCurrentCumulative":   "P",
	"foodLastYearCumulative":  "Q",
	"goodsCurrentMonth":       "R",
	"goodsLastYearMonth":      "S",
	"goodsCurrentCumulative":  "T",
	"goodsLastYearCumulative": "U",
	"retailCurrentMonth":      "V",
	"retailLastYearMonth":     "W",
	"retailCurrentCumulative": "X",
	"retailLastYearCumulative": "Y",
}

func appendWRExcelCoords(g *Graph, id int64, row int, sheet string) {
	if row <= 0 {
		return
	}
	for field, col := range wrExcelColumns {
		node := BuildNodeID(nodeKindWR, id, field)
		g.AddExcelCoord(node, ExcelCoord{Sheet: sheet, Cell: fmt.Sprintf("%s%d", col, row)})
	}
}

func appendACExcelCoords(g *Graph, id int64, row int, sheet string) {
	if row <= 0 {
		return
	}
	for field, col := range acExcelColumns {
		node := BuildNodeID(nodeKindAC, id, field)
		g.AddExcelCoord(node, ExcelCoord{Sheet: sheet, Cell: fmt.Sprintf("%s%d", col, row)})
	}
}

func attachCompanyEdges(g *Graph, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) {
	for _, r := range wrRecords {
		addWRFieldEdges(g, r)
	}
	for _, r := range acRecords {
		addACFieldEdges(g, r)
	}
}

func attachAggregateEdges(g *Graph, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) {
	for _, r := range wrRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindWR, r.ID, field)
		}
		g.AddEdge(n("salesCurrentMonth"), industryNode(r.IndustryType, "salesCurSum"))
		g.AddEdge(n("salesLastYearMonth"), industryNode(r.IndustryType, "salesLastSum"))
		g.AddEdge(n("salesCurrentCumulative"), industryNode(r.IndustryType, "salesCurCumSum"))
		g.AddEdge(n("salesLastYearCumulative"), industryNode(r.IndustryType, "salesLastCumSum"))

		g.AddEdge(n("retailCurrentMonth"), industryNode(r.IndustryType, "retailCurSum"))
		g.AddEdge(n("retailLastYearMonth"), industryNode(r.IndustryType, "retailLastSum"))
		g.AddEdge(n("retailCurrentCumulative"), industryNode(r.IndustryType, "retailCurCumSum"))
		g.AddEdge(n("retailLastYearCumulative"), industryNode(r.IndustryType, "retailLastCumSum"))

		g.AddEdge(n("retailCurrentMonth"), aggregateNode("limitAboveRetailCurSum"))
		g.AddEdge(n("retailLastYearMonth"), aggregateNode("limitAboveRetailLastSum"))
		g.AddEdge(n("retailCurrentCumulative"), aggregateNode("limitAboveRetailCurCumSum"))
		g.AddEdge(n("retailLastYearCumulative"), aggregateNode("limitAboveRetailLastCumSum"))

		if r.IsEatWearUse == 1 {
			g.AddEdge(n("retailCurrentMonth"), aggregateNode("eatWearUseRetailCurSum"))
			g.AddEdge(n("retailLastYearMonth"), aggregateNode("eatWearUseRetailLastSum"))
		}
		if r.IsSmallMicro == 1 {
			g.AddEdge(n("retailCurrentMonth"), aggregateNode("microSmallRetailCurSum"))
			g.AddEdge(n("retailLastYearMonth"), aggregateNode("microSmallRetailLastSum"))
		}
	}

	for _, r := range acRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindAC, r.ID, field)
		}
		g.AddEdge(n("salesCurrentMonth"), industryNode(r.IndustryType, "salesCurSum"))
		g.AddEdge(n("salesLastYearMonth"), industryNode(r.IndustryType, "salesLastSum"))
		g.AddEdge(n("salesCurrentCumulative"), industryNode(r.IndustryType, "salesCurCumSum"))
		g.AddEdge(n("salesLastYearCumulative"), industryNode(r.IndustryType, "salesLastCumSum"))

		g.AddEdge(n("retailCurrentMonth"), industryNode(r.IndustryType, "retailCurSum"))
		g.AddEdge(n("retailLastYearMonth"), industryNode(r.IndustryType, "retailLastSum"))
		g.AddEdge(n("retailCurrentCumulative"), industryNode(r.IndustryType, "retailCurCumSum"))
		g.AddEdge(n("retailLastYearCumulative"), industryNode(r.IndustryType, "retailLastCumSum"))

		g.AddEdge(n("retailCurrentMonth"), aggregateNode("limitAboveRetailCurSum"))
		g.AddEdge(n("retailLastYearMonth"), aggregateNode("limitAboveRetailLastSum"))
		g.AddEdge(n("retailCurrentCumulative"), aggregateNode("limitAboveRetailCurCumSum"))
		g.AddEdge(n("retailLastYearCumulative"), aggregateNode("limitAboveRetailLastCumSum"))
	}
}

func attachReverseEdges(g *Graph, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) {
	for _, r := range wrRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindWR, r.ID, field)
		}
		g.AddReverseEdge(n("salesMonthRate"), n("salesCurrentMonth"))
		g.AddReverseEdge(n("salesMonthRate"), n("salesLastYearMonth"))
		g.AddReverseEdge(n("salesCumulativeRate"), n("salesCurrentCumulative"))
		g.AddReverseEdge(n("salesCumulativeRate"), n("salesLastYearCumulative"))
		g.AddReverseEdge(n("retailMonthRate"), n("retailCurrentMonth"))
		g.AddReverseEdge(n("retailMonthRate"), n("retailLastYearMonth"))
		g.AddReverseEdge(n("retailCumulativeRate"), n("retailCurrentCumulative"))
		g.AddReverseEdge(n("retailCumulativeRate"), n("retailLastYearCumulative"))
	}

	for _, r := range acRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindAC, r.ID, field)
		}
		g.AddReverseEdge(n("salesMonthRate"), n("salesCurrentMonth"))
		g.AddReverseEdge(n("salesMonthRate"), n("salesLastYearMonth"))
		g.AddReverseEdge(n("salesCumulativeRate"), n("salesCurrentCumulative"))
		g.AddReverseEdge(n("salesCumulativeRate"), n("salesLastYearCumulative"))
		g.AddReverseEdge(n("retailMonthRate"), n("retailCurrentMonth"))
		g.AddReverseEdge(n("retailMonthRate"), n("retailLastYearMonth"))
		g.AddReverseEdge(n("retailCumulativeRate"), n("retailCurrentCumulative"))
		g.AddReverseEdge(n("retailCumulativeRate"), n("retailLastYearCumulative"))
		g.AddReverseEdge(n("retailCurrentMonth"), n("foodCurrentMonth"))
		g.AddReverseEdge(n("retailCurrentMonth"), n("goodsCurrentMonth"))
		g.AddReverseEdge(n("retailLastYearMonth"), n("foodLastYearMonth"))
		g.AddReverseEdge(n("retailLastYearMonth"), n("goodsLastYearMonth"))
	}

	attachIndicatorReverseEdges(g, wrRecords, acRecords)
}

func attachIndicatorReverseEdges(g *Graph, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) {
	for _, r := range wrRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindWR, r.ID, field)
		}
		g.AddReverseEdge(indicatorNode("limitAbove_month_value"), n("retailCurrentMonth"))
		g.AddReverseEdge(indicatorNode("limitAbove_month_rate"), n("retailCurrentMonth"))
		g.AddReverseEdge(indicatorNode("limitAbove_cumulative_value"), n("retailCurrentCumulative"))
		g.AddReverseEdge(indicatorNode("limitAbove_cumulative_rate"), n("retailCurrentCumulative"))

		if r.IsEatWearUse == 1 {
			g.AddReverseEdge(indicatorNode("eatWearUse_month_rate"), n("retailCurrentMonth"))
		}
		if r.IsSmallMicro == 1 {
			g.AddReverseEdge(indicatorNode("microSmall_month_rate"), n("retailCurrentMonth"))
		}

		if r.IndustryType == "wholesale" {
			g.AddReverseEdge(indicatorNode("wholesale_month_rate"), n("salesCurrentMonth"))
			g.AddReverseEdge(indicatorNode("wholesale_cumulative_rate"), n("salesCurrentCumulative"))
		}
		if r.IndustryType == "retail" {
			g.AddReverseEdge(indicatorNode("retail_month_rate"), n("salesCurrentMonth"))
			g.AddReverseEdge(indicatorNode("retail_cumulative_rate"), n("salesCurrentCumulative"))
		}
	}

	for _, r := range acRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindAC, r.ID, field)
		}
		g.AddReverseEdge(indicatorNode("limitAbove_month_value"), n("retailCurrentMonth"))
		g.AddReverseEdge(indicatorNode("limitAbove_month_rate"), n("retailCurrentMonth"))
		g.AddReverseEdge(indicatorNode("limitAbove_cumulative_value"), n("retailCurrentCumulative"))
		g.AddReverseEdge(indicatorNode("limitAbove_cumulative_rate"), n("retailCurrentCumulative"))

		if r.IndustryType == "accommodation" {
			g.AddReverseEdge(indicatorNode("accommodation_month_rate"), n("salesCurrentMonth"))
			g.AddReverseEdge(indicatorNode("accommodation_cumulative_rate"), n("salesCurrentCumulative"))
		}
		if r.IndustryType == "catering" {
			g.AddReverseEdge(indicatorNode("catering_month_rate"), n("salesCurrentMonth"))
			g.AddReverseEdge(indicatorNode("catering_cumulative_rate"), n("salesCurrentCumulative"))
		}
	}

	// totalSocial_cumulative_value 上游: limitAbove_cumulative_value, microSmall_month_rate
	g.AddReverseEdge(indicatorNode("totalSocial_cumulative_value"), indicatorNode("limitAbove_cumulative_value"))
	g.AddReverseEdge(indicatorNode("totalSocial_cumulative_value"), indicatorNode("microSmall_month_rate"))

	// totalSocial_cumulative_rate 上游: totalSocial_cumulative_value, limitAboveRetailLastCumSum
	g.AddReverseEdge(indicatorNode("totalSocial_cumulative_rate"), indicatorNode("totalSocial_cumulative_value"))
	g.AddReverseEdge(indicatorNode("totalSocial_cumulative_rate"), aggregateNode("limitAboveRetailLastCumSum"))
}

func attachIndicatorCoords(g *Graph, index *TemplateIndex) {
	whMax := index.maxRows["批发"]
	whGrowthRow := whMax + 2
	totalRow := whGrowthRow + 3
	totalGrowthRow := whGrowthRow + 4

	addIndicatorExcel(g, "limitAbove_month_value", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "G4"},
		{Sheet: "批发", Cell: fmt.Sprintf("J%d", totalRow)},
	})
	addIndicatorExcel(g, "limitAbove_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "S4"},
		{Sheet: "批发", Cell: fmt.Sprintf("K%d", totalGrowthRow)},
	})
	addIndicatorExcel(g, "limitAbove_cumulative_value", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "I4"},
		{Sheet: "批发", Cell: fmt.Sprintf("M%d", totalRow)},
	})
	addIndicatorExcel(g, "limitAbove_cumulative_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "T4"},
		{Sheet: "批发", Cell: fmt.Sprintf("N%d", totalGrowthRow)},
	})

	addIndicatorExcel(g, "wholesale_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "K4"},
		{Sheet: "批发", Cell: fmt.Sprintf("E%d", whGrowthRow)},
	})
	addIndicatorExcel(g, "wholesale_cumulative_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "L4"},
		{Sheet: "批发", Cell: fmt.Sprintf("H%d", whGrowthRow)},
	})

	retailRows := index.maxRows["零售"]
	reGrowthRow := retailRows + 2
	addIndicatorExcel(g, "retail_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "M4"},
		{Sheet: "零售", Cell: fmt.Sprintf("E%d", reGrowthRow)},
	})
	addIndicatorExcel(g, "retail_cumulative_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "N4"},
		{Sheet: "零售", Cell: fmt.Sprintf("H%d", reGrowthRow)},
	})

	accRows := index.maxRows["住宿"]
	accGrowthRow := accRows + 2
	addIndicatorExcel(g, "accommodation_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "O4"},
		{Sheet: "住宿", Cell: fmt.Sprintf("E%d", accGrowthRow)},
	})
	addIndicatorExcel(g, "accommodation_cumulative_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "P4"},
		{Sheet: "住宿", Cell: fmt.Sprintf("H%d", accGrowthRow)},
	})

	catRows := index.maxRows["餐饮"]
	catGrowthRow := catRows + 2
	addIndicatorExcel(g, "catering_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "Q4"},
		{Sheet: "餐饮", Cell: fmt.Sprintf("E%d", catGrowthRow)},
	})
	addIndicatorExcel(g, "catering_cumulative_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "R4"},
		{Sheet: "餐饮", Cell: fmt.Sprintf("H%d", catGrowthRow)},
	})

	addIndicatorExcel(g, "eatWearUse_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "U4"},
		{Sheet: "社零额（定）", Cell: "C4"},
	})
	addIndicatorExcel(g, "microSmall_month_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "V4"},
		{Sheet: "社零额（定）", Cell: "B4"},
	})

	addIndicatorExcel(g, "totalSocial_cumulative_value", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "N10"},
	})
	addIndicatorExcel(g, "totalSocial_cumulative_rate", []ExcelCoord{
		{Sheet: "汇总表（定）", Cell: "S10"},
	})
}

func addIndicatorExcel(g *Graph, id string, coords []ExcelCoord) {
	node := indicatorNode(id)
	for _, c := range coords {
		g.AddExcelCoord(node, c)
	}
}

func attachAggregateCoords(g *Graph, index *TemplateIndex) {
	whMax := index.maxRows["批发"]
	whSumRow := whMax + 1
	whGrowthRow := whMax + 2
	totalRow := whGrowthRow + 3
	totalGrowthRow := whGrowthRow + 4
	attachIndustrySumCoords(g, "wholesale", "批发", whSumRow, whGrowthRow, false)
	addAggregateCoord(g, "limitAboveRetailCurSum", "批发", fmt.Sprintf("J%d", totalRow))
	addAggregateCoord(g, "limitAboveRetailLastSum", "批发", fmt.Sprintf("K%d", totalRow))
	addAggregateCoord(g, "limitAboveRetailCurCumSum", "批发", fmt.Sprintf("M%d", totalRow))
	addAggregateCoord(g, "limitAboveRetailLastCumSum", "批发", fmt.Sprintf("N%d", totalRow))
	addAggregateCoord(g, "limitAboveRetailMonthRate", "批发", fmt.Sprintf("K%d", totalGrowthRow))
	addAggregateCoord(g, "limitAboveRetailCumulativeRate", "批发", fmt.Sprintf("N%d", totalGrowthRow))

	reMax := index.maxRows["零售"]
	reSumRow := reMax + 1
	reGrowthRow := reMax + 2
	attachIndustrySumCoords(g, "retail", "零售", reSumRow, reGrowthRow, false)

	accMax := index.maxRows["住宿"]
	accSumRow := accMax + 1
	accGrowthRow := accMax + 2
	attachIndustrySumCoords(g, "accommodation", "住宿", accSumRow, accGrowthRow, true)

	catMax := index.maxRows["餐饮"]
	catSumRow := catMax + 1
	catGrowthRow := catMax + 2
	attachIndustrySumCoords(g, "catering", "餐饮", catSumRow, catGrowthRow, true)
}

func attachIndustrySumCoords(
	g *Graph,
	industry string,
	sheet string,
	sumRow int,
	growthRow int,
	acStyle bool,
) {
	if acStyle {
		addIndustryCoord(g, industry, "salesCurSum", sheet, fmt.Sprintf("D%d", sumRow))
		addIndustryCoord(g, industry, "salesLastSum", sheet, fmt.Sprintf("E%d", sumRow))
		addIndustryCoord(g, industry, "salesCurCumSum", sheet, fmt.Sprintf("G%d", sumRow))
		addIndustryCoord(g, industry, "salesLastCumSum", sheet, fmt.Sprintf("H%d", sumRow))
		addIndustryCoord(g, industry, "retailCurSum", sheet, fmt.Sprintf("V%d", sumRow))
		addIndustryCoord(g, industry, "retailLastSum", sheet, fmt.Sprintf("W%d", sumRow))
		addIndustryCoord(g, industry, "retailCurCumSum", sheet, fmt.Sprintf("X%d", sumRow))
		addIndustryCoord(g, industry, "retailLastCumSum", sheet, fmt.Sprintf("Y%d", sumRow))
		addIndustryCoord(g, industry, "salesMonthRate", sheet, fmt.Sprintf("E%d", growthRow))
		addIndustryCoord(g, industry, "salesCumulativeRate", sheet, fmt.Sprintf("H%d", growthRow))
		return
	}

	addIndustryCoord(g, industry, "salesCurSum", sheet, fmt.Sprintf("D%d", sumRow))
	addIndustryCoord(g, industry, "salesLastSum", sheet, fmt.Sprintf("E%d", sumRow))
	addIndustryCoord(g, industry, "salesCurCumSum", sheet, fmt.Sprintf("G%d", sumRow))
	addIndustryCoord(g, industry, "salesLastCumSum", sheet, fmt.Sprintf("H%d", sumRow))
	addIndustryCoord(g, industry, "retailCurSum", sheet, fmt.Sprintf("J%d", sumRow))
	addIndustryCoord(g, industry, "retailLastSum", sheet, fmt.Sprintf("K%d", sumRow))
	addIndustryCoord(g, industry, "retailCurCumSum", sheet, fmt.Sprintf("M%d", sumRow))
	addIndustryCoord(g, industry, "retailLastCumSum", sheet, fmt.Sprintf("N%d", sumRow))
	addIndustryCoord(g, industry, "salesMonthRate", sheet, fmt.Sprintf("E%d", growthRow))
	addIndustryCoord(g, industry, "salesCumulativeRate", sheet, fmt.Sprintf("H%d", growthRow))
}

func addIndustryCoord(g *Graph, industry string, field string, sheet string, cell string) {
	g.AddExcelCoord(industryNode(industry, field), ExcelCoord{Sheet: sheet, Cell: cell})
}

func addAggregateCoord(g *Graph, field string, sheet string, cell string) {
	g.AddExcelCoord(aggregateNode(field), ExcelCoord{Sheet: sheet, Cell: cell})
}

func filterWRIndustry(records []*model.WholesaleRetail, industry string) []*model.WholesaleRetail {
	out := make([]*model.WholesaleRetail, 0, len(records))
	for _, r := range records {
		if r.IndustryType == industry {
			out = append(out, r)
		}
	}
	return out
}

func filterACIndustry(records []*model.AccommodationCatering, industry string) []*model.AccommodationCatering {
	out := make([]*model.AccommodationCatering, 0, len(records))
	for _, r := range records {
		if r.IndustryType == industry {
			out = append(out, r)
		}
	}
	return out
}
