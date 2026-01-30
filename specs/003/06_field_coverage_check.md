# 字段覆盖度检查报告

## 检查目标

1. ✅ 能否完整解析输入文件 "12月月报（预估）.xlsx" 的所有字段
2. ✅ 能否完整输出 "12月月报（定）.xlsx" 的所有字段

---

## 1. 输入字段覆盖度检查

### 1.1 批发/零售主表 (输入)

| 序号 | Excel 列名 | 数据库字段 | 状态 | 备注 |
|------|-----------|----------|------|------|
| 1 | 序号 | `row_no` | ✅ | |
| 2 | 统一社会信用代码 | `credit_code` | ✅ | |
| 3 | 单位详细名称 | `name` | ✅ | |
| 4 | [201-1] 行业代码 | `industry_code` | ✅ | |
| 5 | 2025年11月销售额 | `sales_prev_month` | ✅ | |
| 6 | 2025年12月销售额 | `sales_current_month` | ✅ | 可调整 |
| 7 | 2024年;12月;商品销售额 | `sales_last_year_month` | ✅ | |
| 8 | 12月销售额增速 | `sales_month_rate` | ✅ | 计算字段 |
| 9 | 2025年1-11月销售额 | `sales_prev_cumulative` | ✅ | |
| 10 | 2024年1-11月销售额 | `sales_last_year_prev_cumulative` | ✅ | |
| 11 | 2025年1-12月销售额 | `sales_current_cumulative` | ✅ | |
| 12 | 2024年;1-12月;商品销售额 | `sales_last_year_cumulative` | ✅ | |
| 13 | 1-12月增速 | `sales_cumulative_rate` | ✅ | 计算字段 |
| 14 | 2025年11月零售额 | `retail_prev_month` | ✅ | |
| 15 | 2025年12月零售额 | `retail_current_month` | ✅ | 可调整 |
| 16 | 2024年;12月;商品零售额 | `retail_last_year_month` | ✅ | |
| 17 | 12月零售额增速 | `retail_month_rate` | ✅ | 计算字段 |
| 18 | 2025年1-11月零售额 | `retail_prev_cumulative` | ✅ | |
| 19 | 2024年1-11月零售额 | `retail_last_year_prev_cumulative` | ✅ | |
| 20 | 2025年1-12月零售额 | `retail_current_cumulative` | ✅ | |
| 21 | 2024年;1-12月;商品零售额 | `retail_last_year_cumulative` | ✅ | |
| 22 | 1-12月增速.1 | `retail_cumulative_rate` | ✅ | 计算字段 |
| 23 | 零售额占比 | `retail_ratio` | ✅ | 计算字段 |
| 24 | 单位规模 | `company_scale` | ✅ | |
| 25 | 粮油食品类 | `cat_grain_oil_food` | ✅ | |
| 26 | 饮料类 | `cat_beverage` | ✅ | |
| 27 | 烟酒类 | `cat_tobacco_liquor` | ✅ | |
| 28 | 服装鞋帽针纺类 | `cat_clothing` | ✅ | |
| 29 | 日用品类 | `cat_daily_use` | ✅ | |
| 30 | 汽车类 | `cat_automobile` | ✅ | |
| 31 | 小微企业 | `is_small_micro` | ✅ | |
| 32 | 吃穿用 | `is_eat_wear_use` | ✅ | 仅零售 |

**结论**: ✅ 批零主表所有字段都可以解析存储

---

### 1.2 住宿/餐饮主表 (输入)

| 序号 | Excel 列名 | 数据库字段 | 状态 | 备注 |
|------|-----------|----------|------|------|
| 1 | 序号 | `row_no` | ✅ | |
| 2 | 统一社会信用代码 | `credit_code` | ✅ | |
| 3 | 单位详细名称 | `name` | ✅ | |
| 4 | [201-1] 行业代码 | `industry_code` | ✅ | |
| 5 | 2025年11月营业额 | `revenue_prev_month` | ✅ | |
| 6 | 2025年12月营业额 | `revenue_current_month` | ✅ | 可调整 |
| 7 | 2024年12月;营业额总计 | `revenue_last_year_month` | ✅ | |
| 8 | 12月增速 | `revenue_month_rate` | ✅ | 计算字段 |
| 9 | 2025年1-11月营业额 | `revenue_prev_cumulative` | ✅ | |
| 10 | 2025年1-12月营业额 | `revenue_current_cumulative` | ✅ | |
| 11 | 2024年1-12月;营业额总计 | `revenue_last_year_cumulative` | ✅ | |
| 12 | 1-12月增速 | `revenue_cumulative_rate` | ✅ | 计算字段 |
| 13 | 11月客房收入 | `room_prev_month` | ✅ | |
| 14 | 2025年12月客房收入 | `room_current_month` | ✅ | 可调整 |
| 15 | 2024年12月;客房收入 | `room_last_year_month` | ✅ | |
| 16 | 2025年1-11月客房收入 | `room_prev_cumulative` | ✅ | |
| 17 | 2025年1-12月客房收入 | `room_current_cumulative` | ✅ | |
| 18 | 2024年1-12月;客房收入 | `room_last_year_cumulative` | ✅ | |
| 19 | 11月餐费收入 | `food_prev_month` | ✅ | |
| 20 | 2025年12月餐费收入 | `food_current_month` | ✅ | 可调整 |
| 21 | 2024年12月;餐费收入 | `food_last_year_month` | ✅ | |
| 22 | 2025年1-11月餐费收入 | `food_prev_cumulative` | ✅ | |
| 23 | 1-12月餐费收入 | `food_current_cumulative` | ✅ | |
| 24 | 2024年1-12月;餐费收入 | `food_last_year_cumulative` | ✅ | |
| 25 | 11月销售额 | `goods_prev_month` | ✅ | |
| 26 | 2025年12月销售额 | `goods_current_month` | ✅ | 可调整 |
| 27 | 2024年12月;商品销售额 | `goods_last_year_month` | ✅ | |
| 28 | 2025年1-11月销售额 | `goods_prev_cumulative` | ✅ | |
| 29 | 1-12月销售额 | `goods_current_cumulative` | ✅ | |
| 30 | 2024年1-12月;商品销售额 | `goods_last_year_cumulative` | ✅ | |
| 31 | 2025年12月零售额 | `retail_current_month` | ✅ | |
| 32 | 2024年12月零售额 | `retail_last_year_month` | ✅ | |
| 33 | 吃穿用 | `is_eat_wear_use` | ✅ | |
| 34 | 小微企业 | `is_small_micro` | ✅ | 住宿 |

**结论**: ✅ 住餐主表所有字段都可以解析存储

---

### 1.3 历史快照 Sheet (输入)

| Sheet 类型 | 字段覆盖 | 目标表 | 状态 |
|-----------|---------|--------|------|
| 2024年12月批零 | 基础信息 + 销售额 + 零售额 | `wr_snapshot` | ✅ |
| 2025年11月批零 | 同上 + 上年数据 | `wr_snapshot` | ✅ |
| 2024年12月住餐 | 基础信息 + 营业额 + 收入 | `ac_snapshot` | ✅ |
| 2025年11月住餐 | 同上 | `ac_snapshot` | ✅ |

**结论**: ✅ 历史快照字段都可以存储

---

## 2. 输出字段覆盖度检查

### 2.1 批零总表 (输出 - 17 列)

| 序号 | 输出列名 | 数据库字段 | 状态 | 备注 |
|------|---------|----------|------|------|
| 1 | 统一社会信用代码 | `credit_code` | ✅ | |
| 2 | 单位详细名称 | `name` | ✅ | |
| 3 | 行业代码 | `industry_code` | ✅ | |
| 4 | 商品销售额;本年-本月 | `sales_current_month` | ✅ | |
| 5 | 商品销售额;上年-本月 | `sales_last_year_month` | ✅ | |
| 6 | 商品销售额;本年-1—本月 | `sales_current_cumulative` | ✅ | |
| 7 | 商品销售额;上年-1—本月 | `sales_last_year_cumulative` | ✅ | |
| 8 | 零售额;本年-本月 | `retail_current_month` | ✅ | |
| 9 | 零售额;上年-本月 | `retail_last_year_month` | ✅ | |
| 10 | 零售额;本年-1—本月 | `retail_current_cumulative` | ✅ | |
| 11 | 零售额;上年-1—本月 | `retail_last_year_cumulative` | ✅ | |
| 12 | 当月增速（销售额） | `sales_month_rate` | ✅ | 计算 |
| 13 | 累计增速（销售额） | `sales_cumulative_rate` | ✅ | 计算 |
| 14 | 当月增速（零售额） | `retail_month_rate` | ✅ | 计算 |
| 15 | 累计增速（零售额） | `retail_cumulative_rate` | ✅ | 计算 |
| 16 | 第一次上报的IP | ❌ 缺失 | ⚠️ | 需要添加 |
| 17 | 填报IP | ❌ 缺失 | ⚠️ | 需要添加 |

**问题**: 缺少 2 个 IP 字段

---

### 2.2 住餐总表 (输出 - 21 列)

| 字段组 | 数据库字段 | 状态 |
|-------|----------|------|
| 基础信息 (3列) | `credit_code`, `name`, `industry_code` | ✅ |
| 营业额 (4列) | `revenue_*` | ✅ |
| 客房收入 (4列) | `room_*` | ✅ |
| 餐费收入 (4列) | `food_*` | ✅ |
| 商品销售额 (4列) | `goods_*` | ✅ |
| 衍生指标 (2列) | `revenue_month_rate`, `revenue_cumulative_rate` | ✅ |

**备注**: 文档提到"模板中存在 4 个 Unnamed 列"，这些是模板中的空列，导出时需要按模板保留。

**结论**: ✅ 字段完整，但需要特殊处理 Unnamed 列

---

### 2.3 吃穿用 Sheet (输出 - 26 列)

| 字段类型 | 来源 | 状态 | 备注 |
|---------|------|------|------|
| 基础信息 | `credit_code`, `name`, `industry_code` | ✅ | |
| 销售额/零售额 (本年/上年) | `sales_*`, `retail_*` | ✅ | |
| 衍生指标 (增速) | `sales_month_rate`, `retail_month_rate` | ✅ | 计算 |
| 小微计算字段 | - | ⚠️ | 需要计算逻辑 |
| **网络销售额** | ❌ 缺失 | ⚠️ | **需要添加字段** |
| **开业时间年/月** | ❌ 缺失 | ⚠️ | **需要添加字段** |

**问题**:
1. 缺少 `network_sales` (网络销售额) 字段
2. 缺少 `opening_year` 和 `opening_month` (开业时间) 字段

---

### 2.4 小微 Sheet (输出 - 6 列)

| 字段 | 来源 | 状态 |
|------|------|------|
| 基础信息 | 筛选 `is_small_micro = 1` | ✅ |
| 当月零售额 | `retail_current_month` | ✅ |
| 上年同月零售额 | `retail_last_year_month` | ✅ |
| 增速 | `retail_month_rate` | ✅ |

**结论**: ✅ 字段完整

---

### 2.5 社零额（定）Sheet (输出)

这个 Sheet 是纯公式计算，依赖：
- 限上零售额汇总（从主表计算）✅
- 限下估算（从 `config` 表获取）✅
- 增速权重（从 `config` 表获取）✅

**结论**: ✅ 数据来源完整

---

### 2.6 汇总表（定）Sheet (输出)

| 字段类型 | 来源 | 状态 |
|---------|------|------|
| 单位数 | `COUNT(*)` | ✅ |
| 上报率 | `config` 表 | ✅ |
| 当月/累计零售额 | 主表聚合 | ✅ |
| 四大行业增速 | 主表聚合 + 计算 | ✅ |
| 小微/吃穿用增速 | 主表聚合 + 计算 | ✅ |
| 负增长企业数 | `config` 表 | ✅ |

**结论**: ✅ 数据来源完整

---

## 3. 发现的问题汇总

### 3.1 必须添加的字段 ⚠️

#### A. wholesale_retail 表需要添加

```sql
ALTER TABLE wholesale_retail ADD COLUMN first_report_ip TEXT;      -- 第一次上报的IP
ALTER TABLE wholesale_retail ADD COLUMN fill_ip TEXT;              -- 填报IP
ALTER TABLE wholesale_retail ADD COLUMN network_sales REAL;        -- 网络销售额
ALTER TABLE wholesale_retail ADD COLUMN opening_year INTEGER;      -- 开业年份
ALTER TABLE wholesale_retail ADD COLUMN opening_month INTEGER;     -- 开业月份
```

#### B. accommodation_catering 表需要添加

```sql
ALTER TABLE accommodation_catering ADD COLUMN first_report_ip TEXT; -- 第一次上报的IP
ALTER TABLE accommodation_catering ADD COLUMN fill_ip TEXT;         -- 填报IP
ALTER TABLE accommodation_catering ADD COLUMN network_sales REAL;   -- 网络销售额
ALTER TABLE accommodation_catering ADD COLUMN opening_year INTEGER; -- 开业年份
ALTER TABLE accommodation_catering ADD COLUMN opening_month INTEGER;-- 开业月份
```

---

### 3.2 字段用途说明

| 字段名 | 用途 | 来源 | 备注 |
|-------|------|------|------|
| `first_report_ip` | 第一次上报的IP | 输入 Excel（可能） | 批零总表输出需要 |
| `fill_ip` | 填报IP | 输入 Excel（可能） | 批零总表输出需要 |
| `network_sales` | 网络销售额 | 输入 Excel（可能） | 吃穿用 Sheet 输出需要 |
| `opening_year` | 开业年份 | 输入 Excel（可能） | 吃穿用 Sheet 输出需要 |
| `opening_month` | 开业月份 | 输入 Excel（可能） | 吃穿用 Sheet 输出需要 |

---

### 3.3 需要确认的问题 🤔

1. **IP 字段来源**
   - 输入 Excel 中是否有这两个字段？
   - 如果没有，是否需要在导出时留空？

2. **网络销售额字段**
   - 输入 Excel 中是否有此字段？
   - 是否所有企业都有，还是仅部分企业？

3. **开业时间字段**
   - 输入 Excel 中是否有此字段？
   - 格式是什么？（单独的年/月列，还是合并的日期列？）

---

## 4. 兼容性建议

### 4.1 短期方案 (快速上线)

将缺失字段设为可选（允许 NULL），导出时：
- IP 字段：留空
- 网络销售额：留空或填 0
- 开业时间：留空

```sql
-- 添加可选字段
ALTER TABLE wholesale_retail ADD COLUMN first_report_ip TEXT DEFAULT '';
ALTER TABLE wholesale_retail ADD COLUMN fill_ip TEXT DEFAULT '';
ALTER TABLE wholesale_retail ADD COLUMN network_sales REAL DEFAULT 0;
ALTER TABLE wholesale_retail ADD COLUMN opening_year INTEGER;
ALTER TABLE wholesale_retail ADD COLUMN opening_month INTEGER;
```

### 4.2 长期方案 (完整功能)

1. **确认输入 Excel 中是否有这些字段**
2. **如果有**：解析器识别并导入
3. **如果没有**：
   - IP 字段：从系统获取（导入时记录）
   - 网络销售额：提供界面让用户补充
   - 开业时间：提供界面让用户补充

---

## 5. 结论

### ✅ 可以完整解析输入 Excel

- 批零主表：32 个字段 ✅
- 住餐主表：34 个字段 ✅
- 历史快照：完整支持 ✅

### ⚠️ 输出 Excel 缺少 5 个字段

需要添加：
1. `first_report_ip` (批零/住餐)
2. `fill_ip` (批零/住餐)
3. `network_sales` (批零/住餐，吃穿用输出需要)
4. `opening_year` (批零，吃穿用输出需要)
5. `opening_month` (批零，吃穿用输出需要)

### 💡 建议

1. **立即添加这 5 个字段**到表结构中
2. **确认输入 Excel 中是否有这些字段**
3. **如果没有，采用短期方案**（字段留空或填默认值）
4. **长期提供数据补充界面**

---

## 6. 更新后的完整表结构

见下一个文件：`01_database.md` (需要更新)
