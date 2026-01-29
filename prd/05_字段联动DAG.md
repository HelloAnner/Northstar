# 字段联动 DAG 设计

> 目标：任何底层字段变动，能够准确推导出所有受影响的衍生字段与汇总指标。

## 1. DAG 节点分层（建议）

### L0 - 原始输入节点（Input）
**批零类（每企业）**
- 商品销售额;本年-本月
- 商品销售额;上年-本月
- 商品销售额;本年-1—本月
- 商品销售额;上年-1—本月
- 零售额;本年-本月
- 零售额;上年-本月
- 零售额;本年-1—本月
- 零售额;上年-1—本月
- 行业代码 / 单位规模 / 吃穿用标记

**住餐类（每企业）**
- 营业额;本年-本月 / 上年-本月 / 本年-1—本月 / 上年-1—本月
- 客房收入;本年-本月 / 上年-本月 / 本年-1—本月 / 上年-1—本月
- 餐费收入;本年-本月 / 上年-本月 / 本年-1—本月 / 上年-1—本月
- 商品销售额;本年-本月 / 上年-本月 / 本年-1—本月 / 上年-1—本月
- 行业代码 / 单位规模 / 住餐业务属性

**社零额（定）手工输入**
- 当前月份 J2
- 小微/吃穿用/抽样增速（B4/C4/D4, B6/C6/D6）
- 权重（B12/C12/D12）
- 历史累计社零额（E18/E19/E20/E21/E22/E23）
- 全省限下增速变动量（I3）
- 单位数、负增长企业数、备注时间等（汇总表输入项）

---

### L1 - 企业级衍生节点（Per-Company Derived）
- 销售额当月增速 / 累计增速
- 零售额当月增速 / 累计增速
- 营业额当月增速 / 累计增速
- 吃穿用判定（行业码/分类）
- 小微判定（单位规模 3/4）
- 吃穿用当月零售额 / 上年同月零售额
- 小微计算用零售额（当月/上年同月）

---

### L2 - 行业汇总节点（Industry Aggregates）
- 批发销售额：当月 / 上年同期 / 累计 / 上年累计
- 零售销售额：当月 / 上年同期 / 累计 / 上年累计
- 住宿营业额：当月 / 上年同期 / 累计 / 上年累计
- 餐饮营业额：当月 / 上年同期 / 累计 / 上年累计
- 行业增速：当月 / 累计（批/零/住/餐）

---

### L3 - 全局汇总节点（Global Aggregates）
- 限上零售额总计（当月/累计）
- 限上零售额增速（当月/累计）
- 吃穿用增速（当月）
- 小微增速（当月）
- 社零总额（累计）与增速

---

### L4 - 模板输出节点（Output Sheets）
- 批发/零售/住宿/餐饮 数据区
- 批零总表 / 住餐总表
- 吃穿用 / 小微 / 吃穿用（剔除）
- 社零额（定）公式输入区
- 汇总表（定）核心行 + 文案

---

## 2. DAG 依赖关系（简化边）

```
原始字段(企业)
  └─> 企业级衍生(增速、分类)
        ├─> 行业汇总(按行业聚合)
        │     └─> 行业增速
        ├─> 吃穿用汇总
        ├─> 小微汇总
        └─> 限上零售额汇总
              ├─> 社零总额(累计/增速)
              └─> 汇总表/社零额（定）输入
```

---

## 3. DAG 驱动算法（建议）

### 3.1 数据结构
- `nodes`: {nodeId: {deps, value}}
- `reverseDeps`: {nodeId: [children...]}

### 3.2 变更传播流程
1) 用户修改某企业字段（例如“零售额;本年-本月”）
2) 找到受影响节点集合（DFS/BFS 遍历 reverseDeps）
3) 按拓扑序重算（Topological Sort）
4) 仅更新受影响的输出 sheet 区域

### 3.3 增量优化（性能）
- 行业汇总可用“delta 增量”更新
- 汇总表与社零额（定）只重算与该行业相关的指标

---

## 4. Graphviz DAG（核心字段）

```
digraph DataDAG {
  rankdir=LR;
  node [shape=box];

  subgraph cluster_input {
    label="Input";
    c_retail_cur; c_retail_last; c_retail_cur_sum; c_retail_last_sum;
    c_sales_cur; c_sales_last; c_sales_cur_sum; c_sales_last_sum;
    c_rev_cur; c_rev_last; c_rev_cur_sum; c_rev_last_sum;
    c_room_cur; c_room_last; c_food_cur; c_food_last; c_goods_cur; c_goods_last;
    c_code; c_scale;
    manual_social_inputs;
  }

  subgraph cluster_company {
    label="Per-Company Derived";
    rate_retail_month; rate_retail_cum; rate_sales_month; rate_sales_cum;
    rate_rev_month; rate_rev_cum;
    tag_eatwear; tag_micro;
    eatwear_retail_cur; eatwear_retail_last; micro_retail_cur; micro_retail_last;
  }

  subgraph cluster_industry {
    label="Industry Aggregates";
    sum_wh_sales; sum_re_sales; sum_acc_rev; sum_cat_rev;
    rate_wh_sales; rate_re_sales; rate_acc_rev; rate_cat_rev;
  }

  subgraph cluster_global {
    label="Global Aggregates";
    sum_retail_all; rate_retail_all; sum_eatwear; rate_eatwear; sum_micro; rate_micro;
    total_social; rate_total_social;
  }

  subgraph cluster_output {
    label="Outputs";
    sheet_wholesale; sheet_retail; sheet_accom; sheet_catering;
    sheet_br_total; sheet_ac_total; sheet_eatwear; sheet_micro;
    sheet_social; sheet_summary;
  }

  c_retail_cur -> rate_retail_month;
  c_retail_last -> rate_retail_month;
  c_retail_cur_sum -> rate_retail_cum;
  c_retail_last_sum -> rate_retail_cum;

  c_sales_cur -> rate_sales_month;
  c_sales_last -> rate_sales_month;
  c_sales_cur_sum -> rate_sales_cum;
  c_sales_last_sum -> rate_sales_cum;

  c_rev_cur -> rate_rev_month;
  c_rev_last -> rate_rev_month;
  c_rev_cur_sum -> rate_rev_cum;
  c_rev_last_sum -> rate_rev_cum;

  c_code -> tag_eatwear;
  c_scale -> tag_micro;

  c_retail_cur -> eatwear_retail_cur;
  c_retail_last -> eatwear_retail_last;
  tag_eatwear -> eatwear_retail_cur;
  tag_eatwear -> eatwear_retail_last;

  c_retail_cur -> micro_retail_cur;
  c_retail_last -> micro_retail_last;
  tag_micro -> micro_retail_cur;
  tag_micro -> micro_retail_last;

  rate_sales_month -> rate_wh_sales;
  rate_sales_cum -> rate_wh_sales;

  sum_wh_sales -> rate_wh_sales;
  sum_re_sales -> rate_re_sales;
  sum_acc_rev -> rate_acc_rev;
  sum_cat_rev -> rate_cat_rev;

  sum_retail_all -> rate_retail_all;
  sum_eatwear -> rate_eatwear;
  sum_micro -> rate_micro;

  sum_retail_all -> total_social;
  rate_micro -> total_social;
  total_social -> rate_total_social;

  sum_wh_sales -> sheet_wholesale;
  sum_re_sales -> sheet_retail;
  sum_acc_rev -> sheet_accom;
  sum_cat_rev -> sheet_catering;
  sum_retail_all -> sheet_summary;
  rate_retail_all -> sheet_summary;
  rate_wh_sales -> sheet_summary;
  rate_re_sales -> sheet_summary;
  rate_acc_rev -> sheet_summary;
  rate_cat_rev -> sheet_summary;

  manual_social_inputs -> sheet_social;
  sum_retail_all -> sheet_social;
  rate_micro -> sheet_social;
}
```

