/**
 * 联动 DAG 构建细节
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import (
	"fmt"

	"northstar/internal/model"
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

var wrExcelColumns = map[string]string{
	"salesCurrentMonth":      "D",
	"salesLastYearMonth":     "E",
	"salesMonthRate":         "F",
	"salesCurrentCumulative": "G",
	"salesLastYearCumulative": "H",
	"salesCumulativeRate":    "I",
	"retailCurrentMonth":     "J",
	"retailLastYearMonth":    "K",
	"retailMonthRate":        "L",
	"retailCurrentCumulative": "M",
	"retailLastYearCumulative": "N",
	"retailCumulativeRate":   "O",
}

var acExcelColumns = map[string]string{
	"salesCurrentMonth":      "D",
	"salesLastYearMonth":     "E",
	"salesMonthRate":         "F",
	"salesCurrentCumulative": "G",
	"salesLastYearCumulative": "H",
	"salesCumulativeRate":    "I",
	"roomCurrentMonth":       "J",
	"roomLastYearMonth":      "K",
	"roomCurrentCumulative":  "L",
	"roomLastYearCumulative": "M",
	"foodCurrentMonth":       "N",
	"foodLastYearMonth":      "O",
	"foodCurrentCumulative":  "P",
	"foodLastYearCumulative": "Q",
	"goodsCurrentMonth":      "R",
	"goodsLastYearMonth":     "S",
	"goodsCurrentCumulative": "T",
	"goodsLastYearCumulative": "U",
	"retailCurrentMonth":     "V",
	"retailLastYearMonth":    "W",
	"retailCurrentCumulative": "X",
	"retailLastYearCumulative": "Y",
}

func attachIndicatorNodes(g *Graph) {
	for _, id := range indicatorIDs {
		node := indicatorNode(id)
		g.IndicatorIDs[node] = id
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
			g.addUICoord(BuildNodeID(nodeKindWR, r.ID, key), UICoord{RowID: rowID, ColumnKey: key})
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
			g.addUICoord(BuildNodeID(nodeKindAC, r.ID, key), UICoord{RowID: rowID, ColumnKey: key})
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

func appendWRExcelCoords(g *Graph, id int64, row int, sheet string) {
	if row <= 0 {
		return
	}
	for field, col := range wrExcelColumns {
		node := BuildNodeID(nodeKindWR, id, field)
		g.addExcelCoord(node, ExcelCoord{Sheet: sheet, Cell: fmt.Sprintf("%s%d", col, row)})
	}
}

func appendACExcelCoords(g *Graph, id int64, row int, sheet string) {
	if row <= 0 {
		return
	}
	for field, col := range acExcelColumns {
		node := BuildNodeID(nodeKindAC, id, field)
		g.addExcelCoord(node, ExcelCoord{Sheet: sheet, Cell: fmt.Sprintf("%s%d", col, row)})
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
		g.addEdge(n("salesCurrentMonth"), industryNode(r.IndustryType, "salesCurSum"))
		g.addEdge(n("salesLastYearMonth"), industryNode(r.IndustryType, "salesLastSum"))
		g.addEdge(n("salesCurrentCumulative"), industryNode(r.IndustryType, "salesCurCumSum"))
		g.addEdge(n("salesLastYearCumulative"), industryNode(r.IndustryType, "salesLastCumSum"))

		g.addEdge(n("retailCurrentMonth"), industryNode(r.IndustryType, "retailCurSum"))
		g.addEdge(n("retailLastYearMonth"), industryNode(r.IndustryType, "retailLastSum"))
		g.addEdge(n("retailCurrentCumulative"), industryNode(r.IndustryType, "retailCurCumSum"))
		g.addEdge(n("retailLastYearCumulative"), industryNode(r.IndustryType, "retailLastCumSum"))

		g.addEdge(n("retailCurrentMonth"), aggregateNode("limitAboveRetailCurSum"))
		g.addEdge(n("retailLastYearMonth"), aggregateNode("limitAboveRetailLastSum"))
		g.addEdge(n("retailCurrentCumulative"), aggregateNode("limitAboveRetailCurCumSum"))
		g.addEdge(n("retailLastYearCumulative"), aggregateNode("limitAboveRetailLastCumSum"))

		if r.IsEatWearUse == 1 {
			g.addEdge(n("retailCurrentMonth"), aggregateNode("eatWearUseRetailCurSum"))
			g.addEdge(n("retailLastYearMonth"), aggregateNode("eatWearUseRetailLastSum"))
		}
		if r.IsSmallMicro == 1 {
			g.addEdge(n("retailCurrentMonth"), aggregateNode("microSmallRetailCurSum"))
			g.addEdge(n("retailLastYearMonth"), aggregateNode("microSmallRetailLastSum"))
		}
	}

	for _, r := range acRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindAC, r.ID, field)
		}
		g.addEdge(n("salesCurrentMonth"), industryNode(r.IndustryType, "salesCurSum"))
		g.addEdge(n("salesLastYearMonth"), industryNode(r.IndustryType, "salesLastSum"))
		g.addEdge(n("salesCurrentCumulative"), industryNode(r.IndustryType, "salesCurCumSum"))
		g.addEdge(n("salesLastYearCumulative"), industryNode(r.IndustryType, "salesLastCumSum"))

		g.addEdge(n("retailCurrentMonth"), industryNode(r.IndustryType, "retailCurSum"))
		g.addEdge(n("retailLastYearMonth"), industryNode(r.IndustryType, "retailLastSum"))
		g.addEdge(n("retailCurrentCumulative"), industryNode(r.IndustryType, "retailCurCumSum"))
		g.addEdge(n("retailLastYearCumulative"), industryNode(r.IndustryType, "retailLastCumSum"))

		g.addEdge(n("retailCurrentMonth"), aggregateNode("limitAboveRetailCurSum"))
		g.addEdge(n("retailLastYearMonth"), aggregateNode("limitAboveRetailLastSum"))
		g.addEdge(n("retailCurrentCumulative"), aggregateNode("limitAboveRetailCurCumSum"))
		g.addEdge(n("retailLastYearCumulative"), aggregateNode("limitAboveRetailLastCumSum"))
	}
}

func attachReverseEdges(g *Graph, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) {
	for _, r := range wrRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindWR, r.ID, field)
		}
		g.addReverseEdge(n("salesMonthRate"), n("salesCurrentMonth"))
		g.addReverseEdge(n("salesMonthRate"), n("salesLastYearMonth"))
		g.addReverseEdge(n("salesCumulativeRate"), n("salesCurrentCumulative"))
		g.addReverseEdge(n("salesCumulativeRate"), n("salesLastYearCumulative"))
		g.addReverseEdge(n("retailMonthRate"), n("retailCurrentMonth"))
		g.addReverseEdge(n("retailMonthRate"), n("retailLastYearMonth"))
		g.addReverseEdge(n("retailCumulativeRate"), n("retailCurrentCumulative"))
		g.addReverseEdge(n("retailCumulativeRate"), n("retailLastYearCumulative"))
	}

	for _, r := range acRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindAC, r.ID, field)
		}
		g.addReverseEdge(n("salesMonthRate"), n("salesCurrentMonth"))
		g.addReverseEdge(n("salesMonthRate"), n("salesLastYearMonth"))
		g.addReverseEdge(n("salesCumulativeRate"), n("salesCurrentCumulative"))
		g.addReverseEdge(n("salesCumulativeRate"), n("salesLastYearCumulative"))
		g.addReverseEdge(n("retailCurrentMonth"), n("foodCurrentMonth"))
		g.addReverseEdge(n("retailCurrentMonth"), n("goodsCurrentMonth"))
		g.addReverseEdge(n("retailLastYearMonth"), n("foodLastYearMonth"))
		g.addReverseEdge(n("retailLastYearMonth"), n("goodsLastYearMonth"))
	}

	attachIndicatorReverseEdges(g, wrRecords, acRecords)
}

func attachIndicatorReverseEdges(g *Graph, wrRecords []*model.WholesaleRetail, acRecords []*model.AccommodationCatering) {
	for _, r := range wrRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindWR, r.ID, field)
		}
		g.addReverseEdge(indicatorNode("limitAbove_month_value"), n("retailCurrentMonth"))
		g.addReverseEdge(indicatorNode("limitAbove_month_rate"), n("retailCurrentMonth"))
		g.addReverseEdge(indicatorNode("limitAbove_cumulative_value"), n("retailCurrentCumulative"))
		g.addReverseEdge(indicatorNode("limitAbove_cumulative_rate"), n("retailCurrentCumulative"))

		if r.IsEatWearUse == 1 {
			g.addReverseEdge(indicatorNode("eatWearUse_month_rate"), n("retailCurrentMonth"))
		}
		if r.IsSmallMicro == 1 {
			g.addReverseEdge(indicatorNode("microSmall_month_rate"), n("retailCurrentMonth"))
		}

		if r.IndustryType == "wholesale" {
			g.addReverseEdge(indicatorNode("wholesale_month_rate"), n("salesCurrentMonth"))
			g.addReverseEdge(indicatorNode("wholesale_cumulative_rate"), n("salesCurrentCumulative"))
		}
		if r.IndustryType == "retail" {
			g.addReverseEdge(indicatorNode("retail_month_rate"), n("salesCurrentMonth"))
			g.addReverseEdge(indicatorNode("retail_cumulative_rate"), n("salesCurrentCumulative"))
		}
	}

	for _, r := range acRecords {
		n := func(field string) NodeID {
			return BuildNodeID(nodeKindAC, r.ID, field)
		}
		g.addReverseEdge(indicatorNode("limitAbove_month_value"), n("retailCurrentMonth"))
		g.addReverseEdge(indicatorNode("limitAbove_month_rate"), n("retailCurrentMonth"))
		g.addReverseEdge(indicatorNode("limitAbove_cumulative_value"), n("retailCurrentCumulative"))
		g.addReverseEdge(indicatorNode("limitAbove_cumulative_rate"), n("retailCurrentCumulative"))

		if r.IndustryType == "accommodation" {
			g.addReverseEdge(indicatorNode("accommodation_month_rate"), n("salesCurrentMonth"))
			g.addReverseEdge(indicatorNode("accommodation_cumulative_rate"), n("salesCurrentCumulative"))
		}
		if r.IndustryType == "catering" {
			g.addReverseEdge(indicatorNode("catering_month_rate"), n("salesCurrentMonth"))
			g.addReverseEdge(indicatorNode("catering_cumulative_rate"), n("salesCurrentCumulative"))
		}
	}

	for _, id := range []string{"totalSocial_cumulative_value", "totalSocial_cumulative_rate"} {
		node := indicatorNode(id)
		g.addReverseEdge(node, indicatorNode("limitAbove_cumulative_value"))
		g.addReverseEdge(node, indicatorNode("limitAbove_cumulative_rate"))
	}
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
		g.addExcelCoord(node, c)
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
	g.addExcelCoord(industryNode(industry, field), ExcelCoord{Sheet: sheet, Cell: cell})
}

func addAggregateCoord(g *Graph, field string, sheet string, cell string) {
	g.addExcelCoord(aggregateNode(field), ExcelCoord{Sheet: sheet, Cell: cell})
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
