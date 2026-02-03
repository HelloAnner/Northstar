/**
 * 联动 DAG 边与节点规则
 *
 * @author Anner
 * Created on 2026/2/3
 */

package linkage

import "northstar/internal/model"

const (
	nodeKindWR        = "wr"
	nodeKindAC        = "ac"
	nodeKindIndicator = "indicator"
)

func indicatorNode(id string) NodeID {
	return NodeID(nodeKindIndicator + ":" + id)
}

func industryNode(industry string, field string) NodeID {
	return NodeID("industry:" + industry + ":" + field)
}

func aggregateNode(field string) NodeID {
	return NodeID("aggregate:" + field)
}

func addWRFieldEdges(g *Graph, r *model.WholesaleRetail) {
	n := func(field string) NodeID {
		return BuildNodeID(nodeKindWR, r.ID, field)
	}

	addSalesEdges(g, n)
	addRetailEdges(g, n)

	g.addEdge(n("salesCurrentMonth"), n("salesCurrentCumulative"))
	g.addEdge(n("salesLastYearMonth"), n("salesLastYearCumulative"))
	g.addEdge(n("retailCurrentMonth"), n("retailCurrentCumulative"))
	g.addEdge(n("retailLastYearMonth"), n("retailLastYearCumulative"))
}

func addACFieldEdges(g *Graph, r *model.AccommodationCatering) {
	n := func(field string) NodeID {
		return BuildNodeID(nodeKindAC, r.ID, field)
	}

	addSalesEdges(g, n)
	addRetailEdges(g, n)

	g.addEdge(n("foodCurrentMonth"), n("retailCurrentMonth"))
	g.addEdge(n("goodsCurrentMonth"), n("retailCurrentMonth"))
	g.addEdge(n("foodLastYearMonth"), n("retailLastYearMonth"))
	g.addEdge(n("goodsLastYearMonth"), n("retailLastYearMonth"))
	g.addEdge(n("foodCurrentCumulative"), n("retailCurrentCumulative"))
	g.addEdge(n("goodsCurrentCumulative"), n("retailCurrentCumulative"))
	g.addEdge(n("foodLastYearCumulative"), n("retailLastYearCumulative"))
	g.addEdge(n("goodsLastYearCumulative"), n("retailLastYearCumulative"))

	g.addEdge(n("salesCurrentMonth"), n("salesCurrentCumulative"))
	g.addEdge(n("salesLastYearMonth"), n("salesLastYearCumulative"))
}

func addSalesEdges(g *Graph, n func(string) NodeID) {
	g.addEdge(n("salesCurrentMonth"), n("salesMonthRate"))
	g.addEdge(n("salesLastYearMonth"), n("salesMonthRate"))
	g.addEdge(n("salesCurrentMonth"), n("salesYoYDiff"))
	g.addEdge(n("salesLastYearMonth"), n("salesYoYDiff"))
	g.addEdge(n("salesCurrentMonth"), n("salesMoMDiff"))
	g.addEdge(n("salesPrevMonth"), n("salesMoMDiff"))
	g.addEdge(n("salesCurrentMonth"), n("salesMoMRate"))
	g.addEdge(n("salesPrevMonth"), n("salesMoMRate"))
	g.addEdge(n("salesCurrentCumulative"), n("salesCumulativeRate"))
	g.addEdge(n("salesLastYearCumulative"), n("salesCumulativeRate"))
	g.addEdge(n("salesCurrentCumulative"), n("salesCumulativeYoYDiff"))
	g.addEdge(n("salesLastYearCumulative"), n("salesCumulativeYoYDiff"))
}

func addRetailEdges(g *Graph, n func(string) NodeID) {
	g.addEdge(n("retailCurrentMonth"), n("retailMonthRate"))
	g.addEdge(n("retailLastYearMonth"), n("retailMonthRate"))
	g.addEdge(n("retailCurrentMonth"), n("retailYoYDiff"))
	g.addEdge(n("retailLastYearMonth"), n("retailYoYDiff"))
	g.addEdge(n("retailCurrentMonth"), n("retailMoMDiff"))
	g.addEdge(n("retailPrevMonth"), n("retailMoMDiff"))
	g.addEdge(n("retailCurrentMonth"), n("retailMoMRate"))
	g.addEdge(n("retailPrevMonth"), n("retailMoMRate"))
	g.addEdge(n("retailCurrentCumulative"), n("retailCumulativeRate"))
	g.addEdge(n("retailLastYearCumulative"), n("retailCumulativeRate"))
	g.addEdge(n("retailCurrentCumulative"), n("retailCumulativeYoYDiff"))
	g.addEdge(n("retailLastYearCumulative"), n("retailCumulativeYoYDiff"))
	g.addEdge(n("retailCurrentMonth"), n("retailRatio"))
	g.addEdge(n("salesCurrentMonth"), n("retailRatio"))
}

func addIndicatorEdges(g *Graph) {
	g.addEdge(aggregateNode("limitAboveRetailCurSum"), indicatorNode("limitAbove_month_value"))
	g.addEdge(aggregateNode("limitAboveRetailCurSum"), indicatorNode("limitAbove_month_rate"))
	g.addEdge(aggregateNode("limitAboveRetailLastSum"), indicatorNode("limitAbove_month_rate"))

	g.addEdge(aggregateNode("limitAboveRetailCurCumSum"), indicatorNode("limitAbove_cumulative_value"))
	g.addEdge(aggregateNode("limitAboveRetailCurCumSum"), indicatorNode("limitAbove_cumulative_rate"))
	g.addEdge(aggregateNode("limitAboveRetailLastCumSum"), indicatorNode("limitAbove_cumulative_rate"))

	g.addEdge(aggregateNode("eatWearUseRetailCurSum"), indicatorNode("eatWearUse_month_rate"))
	g.addEdge(aggregateNode("eatWearUseRetailLastSum"), indicatorNode("eatWearUse_month_rate"))

	g.addEdge(aggregateNode("microSmallRetailCurSum"), indicatorNode("microSmall_month_rate"))
	g.addEdge(aggregateNode("microSmallRetailLastSum"), indicatorNode("microSmall_month_rate"))

	g.addEdge(industryNode("wholesale", "salesCurSum"), indicatorNode("wholesale_month_rate"))
	g.addEdge(industryNode("wholesale", "salesLastSum"), indicatorNode("wholesale_month_rate"))
	g.addEdge(industryNode("wholesale", "salesCurCumSum"), indicatorNode("wholesale_cumulative_rate"))
	g.addEdge(industryNode("wholesale", "salesLastCumSum"), indicatorNode("wholesale_cumulative_rate"))

	g.addEdge(industryNode("retail", "salesCurSum"), indicatorNode("retail_month_rate"))
	g.addEdge(industryNode("retail", "salesLastSum"), indicatorNode("retail_month_rate"))
	g.addEdge(industryNode("retail", "salesCurCumSum"), indicatorNode("retail_cumulative_rate"))
	g.addEdge(industryNode("retail", "salesLastCumSum"), indicatorNode("retail_cumulative_rate"))

	g.addEdge(industryNode("accommodation", "salesCurSum"), indicatorNode("accommodation_month_rate"))
	g.addEdge(industryNode("accommodation", "salesLastSum"), indicatorNode("accommodation_month_rate"))
	g.addEdge(industryNode("accommodation", "salesCurCumSum"), indicatorNode("accommodation_cumulative_rate"))
	g.addEdge(industryNode("accommodation", "salesLastCumSum"), indicatorNode("accommodation_cumulative_rate"))

	g.addEdge(industryNode("catering", "salesCurSum"), indicatorNode("catering_month_rate"))
	g.addEdge(industryNode("catering", "salesLastSum"), indicatorNode("catering_month_rate"))
	g.addEdge(industryNode("catering", "salesCurCumSum"), indicatorNode("catering_cumulative_rate"))
	g.addEdge(industryNode("catering", "salesLastCumSum"), indicatorNode("catering_cumulative_rate"))

	g.addEdge(indicatorNode("limitAbove_cumulative_value"), indicatorNode("totalSocial_cumulative_value"))
	g.addEdge(indicatorNode("microSmall_month_rate"), indicatorNode("totalSocial_cumulative_value"))
	g.addEdge(indicatorNode("totalSocial_cumulative_value"), indicatorNode("totalSocial_cumulative_rate"))
	g.addEdge(aggregateNode("limitAboveRetailLastCumSum"), indicatorNode("totalSocial_cumulative_rate"))
}
