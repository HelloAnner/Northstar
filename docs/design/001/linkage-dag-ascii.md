# Linkage Graph (ASCII)

Author: Anner
Created: 2026-02-03
Purpose: ASCII-only linkage graph for the preview DAG (detail -> aggregate -> indicators).

Legend:
  ->  forward dependency (normal compute)
  <-  reverse dependency (preview highlight)
  []  DAG node

1) Detail-level (WR/AC)

[UI cell rowId|columnKey]
        |
        v
[wr:{id}:{field}] or [ac:{id}:{field}]
        |
        +--> [derived fields] (YoYDiff/MoMDiff/MoMRate/MonthRate/CumRate/CumYoYDiff/RetailRatio)
        |
        +--> [industry:{type}:salesCurSum / salesLastSum / salesCurCumSum / salesLastCumSum]
        |
        +--> [industry:{type}:retailCurSum / retailLastSum / retailCurCumSum / retailLastCumSum]
        |
        +--> [aggregate:limitAboveRetailCurSum / limitAboveRetailLastSum]
        |
        +--> [aggregate:limitAboveRetailCurCumSum / limitAboveRetailLastCumSum]

AC split (retail is composed of food + goods):

[ac:{id}:foodCurrentMonth]  \
                           +--> [ac:{id}:retailCurrentMonth]
[ac:{id}:goodsCurrentMonth] /

[ac:{id}:foodLastYearMonth]  \
                            +--> [ac:{id}:retailLastYearMonth]
[ac:{id}:goodsLastYearMonth] /

[ac:{id}:foodCurrentCumulative]  \
                                 +--> [ac:{id}:retailCurrentCumulative]
[ac:{id}:goodsCurrentCumulative] /

[ac:{id}:foodLastYearCumulative]  \
                                  +--> [ac:{id}:retailLastYearCumulative]
[ac:{id}:goodsLastYearCumulative] /

Reverse preview edges (for highlighting inputs when clicking derived/output):

[wr:{id}:salesMonthRate]    <- [wr:{id}:salesCurrentMonth]
                           <- [wr:{id}:salesLastYearMonth]

[wr:{id}:salesCumulativeRate] <- [wr:{id}:salesCurrentCumulative]
                              <- [wr:{id}:salesLastYearCumulative]

[wr:{id}:retailMonthRate]   <- [wr:{id}:retailCurrentMonth]
                           <- [wr:{id}:retailLastYearMonth]

[ac:{id}:retailCurrentMonth] <- [ac:{id}:foodCurrentMonth]
                             <- [ac:{id}:goodsCurrentMonth]

2) Aggregates -> Indicators

[industry:{type}:salesCurSum]  \
                               +--> [indicator:{industry}_month_rate]
[industry:{type}:salesLastSum] /

[industry:{type}:salesCurCumSum]  \
                                  +--> [indicator:{industry}_cumulative_rate]
[industry:{type}:salesLastCumSum] /

[aggregate:limitAboveRetailCurSum]  \
                                    +--> [indicator:limitAbove_month_value]
[aggregate:limitAboveRetailLastSum] /     [indicator:limitAbove_month_rate]

[aggregate:limitAboveRetailCurCumSum]  \
                                       +--> [indicator:limitAbove_cumulative_value]
[aggregate:limitAboveRetailLastCumSum] /     [indicator:limitAbove_cumulative_rate]

[aggregate:eatWearUseRetailCurSum]  \
                                    +--> [indicator:eatWearUse_month_rate]
[aggregate:eatWearUseRetailLastSum] /

[aggregate:microSmallRetailCurSum]  \
                                    +--> [indicator:microSmall_month_rate]
[aggregate:microSmallRetailLastSum] /

3) Total social indicators

[indicator:limitAbove_cumulative_value] -> [indicator:totalSocial_cumulative_value]
[indicator:microSmall_month_rate]       -> [indicator:totalSocial_cumulative_value]

[indicator:totalSocial_cumulative_value] -> [indicator:totalSocial_cumulative_rate]
[aggregate:limitAboveRetailLastCumSum]   -> [indicator:totalSocial_cumulative_rate]

Reverse preview edges (indicator -> detail):

[indicator:limitAbove_month_value] <- [wr:{id}:retailCurrentMonth]
[indicator:limitAbove_month_rate]  <- [wr:{id}:retailCurrentMonth]
[indicator:limitAbove_cumulative_value] <- [wr:{id}:retailCurrentCumulative]
[indicator:limitAbove_cumulative_rate]  <- [wr:{id}:retailCurrentCumulative]

[indicator:wholesale_month_rate] <- [wr:{id}:salesCurrentMonth]
[indicator:wholesale_cumulative_rate] <- [wr:{id}:salesCurrentCumulative]
[indicator:retail_month_rate] <- [wr:{id}:salesCurrentMonth]
[indicator:retail_cumulative_rate] <- [wr:{id}:salesCurrentCumulative]

[indicator:accommodation_month_rate] <- [ac:{id}:salesCurrentMonth]
[indicator:accommodation_cumulative_rate] <- [ac:{id}:salesCurrentCumulative]
[indicator:catering_month_rate] <- [ac:{id}:salesCurrentMonth]
[indicator:catering_cumulative_rate] <- [ac:{id}:salesCurrentCumulative]

[indicator:totalSocial_cumulative_value] <- [indicator:limitAbove_cumulative_value]
[indicator:totalSocial_cumulative_rate]  <- [indicator:limitAbove_cumulative_rate]

4) Preview traversal (BFS over forward + reverse edges)

[anchor node]
    |
    v
[BFS over (Edges + ReverseEdges)] -> [Impact nodes with UI/Excel/Indicator coords]
