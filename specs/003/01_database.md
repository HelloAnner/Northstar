# 数据库表结构设计

## 设计原则

1. **全量入库**：Excel 的每个 Sheet、每个字段都能解析存储
2. **分表存储**：批零企业和住餐企业字段差异大，分表处理
3. **历史快照**：支持存储历史月份的快照数据
4. **灵活扩展**：未来新增 Sheet 类型可快速支持

## 数据库文件

```
data/northstar.db  (SQLite)
```

---

## 表结构总览

| 表名 | 说明 | 数据来源 |
|------|------|----------|
| `wholesale_retail` | 批发零售企业 | 批发/零售 Sheet |
| `accommodation_catering` | 住宿餐饮企业 | 住宿/餐饮 Sheet |
| `wr_snapshot` | 批零历史快照 | 2024年12月批零 等 |
| `ac_snapshot` | 住餐历史快照 | 2024年12月住餐 等 |
| `sheets_meta` | Sheet 元信息 | 记录导入的所有 Sheet |
| `config` | 系统配置 | 手工输入项 |
| `import_logs` | 导入日志 | 操作记录 |

---

## 1. wholesale_retail - 批发零售企业表

存储批发、零售 Sheet 的企业数据。

```sql
CREATE TABLE wholesale_retail (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 基础信息 ===
    credit_code TEXT,                            -- 统一社会信用代码
    name TEXT NOT NULL,                          -- 单位详细名称
    industry_code TEXT,                          -- [201-1] 行业代码(GB/T4754-2017)
    industry_type TEXT,                          -- 行业类型: wholesale/retail
    company_scale INTEGER,                       -- 单位规模 (1/2/3/4, 3/4为小微)
    row_no INTEGER,                              -- 原始行号

    -- === 数据月份标识 ===
    data_year INTEGER NOT NULL,                  -- 数据年份 (如 2025)
    data_month INTEGER NOT NULL,                 -- 数据月份 (如 12)

    -- === 销售额 (商品销售额) ===
    sales_prev_month REAL DEFAULT 0,             -- 上月销售额 (如2025年11月)
    sales_current_month REAL DEFAULT 0,          -- 本月销售额 (如2025年12月) ★可调整
    sales_last_year_month REAL DEFAULT 0,        -- 上年同期 (如2024年12月)
    sales_month_rate REAL,                       -- 当月销售额增速 (计算)
    sales_prev_cumulative REAL DEFAULT 0,        -- 本年累计到上月 (如2025年1-11月)
    sales_last_year_prev_cumulative REAL DEFAULT 0, -- 上年累计到上月 (如2024年1-11月)
    sales_current_cumulative REAL DEFAULT 0,     -- 本年累计 (如2025年1-12月)
    sales_last_year_cumulative REAL DEFAULT 0,   -- 上年累计 (如2024年1-12月)
    sales_cumulative_rate REAL,                  -- 累计增速 (计算)

    -- === 零售额 ===
    retail_prev_month REAL DEFAULT 0,            -- 上月零售额 (如2025年11月)
    retail_current_month REAL DEFAULT 0,         -- 本月零售额 (如2025年12月) ★可调整
    retail_last_year_month REAL DEFAULT 0,       -- 上年同期 (如2024年12月)
    retail_month_rate REAL,                      -- 当月零售额增速 (计算)
    retail_prev_cumulative REAL DEFAULT 0,       -- 本年累计到上月 (如2025年1-11月)
    retail_last_year_prev_cumulative REAL DEFAULT 0, -- 上年累计到上月 (如2024年1-11月)
    retail_current_cumulative REAL DEFAULT 0,    -- 本年累计 (如2025年1-12月)
    retail_last_year_cumulative REAL DEFAULT 0,  -- 上年累计 (如2024年1-12月)
    retail_cumulative_rate REAL,                 -- 累计增速 (计算)
    retail_ratio REAL,                           -- 零售额占比 (零销比)

    -- === 商品分类销售额 ===
    cat_grain_oil_food REAL DEFAULT 0,           -- 粮油食品类
    cat_beverage REAL DEFAULT 0,                 -- 饮料类
    cat_tobacco_liquor REAL DEFAULT 0,           -- 烟酒类
    cat_clothing REAL DEFAULT 0,                 -- 服装鞋帽针纺类
    cat_daily_use REAL DEFAULT 0,                -- 日用品类
    cat_automobile REAL DEFAULT 0,               -- 汽车类

    -- === 分类标记 ===
    is_small_micro INTEGER DEFAULT 0,            -- 小微企业标记 (计算: scale=3/4)
    is_eat_wear_use INTEGER DEFAULT 0,           -- 吃穿用标记

    -- === 补充字段 (输出定稿需要) ===
    first_report_ip TEXT,                        -- 第一次上报的IP
    fill_ip TEXT,                                -- 填报IP
    network_sales REAL DEFAULT 0,                -- 网络销售额
    opening_year INTEGER,                        -- 开业年份
    opening_month INTEGER,                       -- 开业月份

    -- === 原始值备份 (用于重置) ===
    original_sales_current_month REAL,
    original_retail_current_month REAL,

    -- === 元数据 ===
    source_sheet TEXT,                           -- 来源 Sheet 名
    source_file TEXT,                            -- 来源文件名
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_wr_data_month ON wholesale_retail(data_year, data_month);
CREATE INDEX idx_wr_credit_code ON wholesale_retail(credit_code);
CREATE INDEX idx_wr_industry_type ON wholesale_retail(industry_type);
CREATE INDEX idx_wr_company_scale ON wholesale_retail(company_scale);
CREATE INDEX idx_wr_is_small_micro ON wholesale_retail(is_small_micro);
CREATE INDEX idx_wr_is_eat_wear_use ON wholesale_retail(is_eat_wear_use);
```

---

## 2. accommodation_catering - 住宿餐饮企业表

存储住宿、餐饮 Sheet 的企业数据。

```sql
CREATE TABLE accommodation_catering (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 基础信息 ===
    credit_code TEXT,                            -- 统一社会信用代码
    name TEXT NOT NULL,                          -- 单位详细名称
    industry_code TEXT,                          -- [201-1] 行业代码(GB/T4754-2017)
    industry_type TEXT,                          -- 行业类型: accommodation/catering
    company_scale INTEGER,                       -- 单位规模
    row_no INTEGER,                              -- 原始行号

    -- === 数据月份标识 ===
    data_year INTEGER NOT NULL,                  -- 数据年份 (如 2025)
    data_month INTEGER NOT NULL,                 -- 数据月份 (如 12)

    -- === 营业额 ===
    revenue_prev_month REAL DEFAULT 0,           -- 上月营业额 (如2025年11月)
    revenue_current_month REAL DEFAULT 0,        -- 本月营业额 (如2025年12月) ★可调整
    revenue_last_year_month REAL DEFAULT 0,      -- 上年同期 (如2024年12月)
    revenue_month_rate REAL,                     -- 当月增速 (计算)
    revenue_prev_cumulative REAL DEFAULT 0,      -- 本年累计到上月 (如2025年1-11月)
    revenue_current_cumulative REAL DEFAULT 0,   -- 本年累计 (如2025年1-12月)
    revenue_last_year_cumulative REAL DEFAULT 0, -- 上年累计 (如2024年1-12月)
    revenue_cumulative_rate REAL,                -- 累计增速 (计算)

    -- === 客房收入 ===
    room_prev_month REAL DEFAULT 0,              -- 上月客房收入 (如2025年11月)
    room_current_month REAL DEFAULT 0,           -- 本月客房收入 (如2025年12月) ★可调整
    room_last_year_month REAL DEFAULT 0,         -- 上年同期客房收入 (如2024年12月)
    room_prev_cumulative REAL DEFAULT 0,         -- 本年累计到上月 (如2025年1-11月)
    room_current_cumulative REAL DEFAULT 0,      -- 本年累计 (如2025年1-12月)
    room_last_year_cumulative REAL DEFAULT 0,    -- 上年累计 (如2024年1-12月)

    -- === 餐费收入 ===
    food_prev_month REAL DEFAULT 0,              -- 上月餐费收入 (如2025年11月)
    food_current_month REAL DEFAULT 0,           -- 本月餐费收入 (如2025年12月) ★可调整
    food_last_year_month REAL DEFAULT 0,         -- 上年同期餐费收入 (如2024年12月)
    food_prev_cumulative REAL DEFAULT 0,         -- 本年累计到上月 (如2025年1-11月)
    food_current_cumulative REAL DEFAULT 0,      -- 本年累计 (如2025年1-12月)
    food_last_year_cumulative REAL DEFAULT 0,    -- 上年累计 (如2024年1-12月)

    -- === 商品销售额 ===
    goods_prev_month REAL DEFAULT 0,             -- 上月销售额 (如2025年11月)
    goods_current_month REAL DEFAULT 0,          -- 本月销售额 (如2025年12月) ★可调整
    goods_last_year_month REAL DEFAULT 0,        -- 上年同期商品销售额 (如2024年12月)
    goods_prev_cumulative REAL DEFAULT 0,        -- 本年累计到上月 (如2025年1-11月)
    goods_current_cumulative REAL DEFAULT 0,     -- 本年累计 (如2025年1-12月)
    goods_last_year_cumulative REAL DEFAULT 0,   -- 上年累计 (如2024年1-12月)

    -- === 零售额 (住餐也有) ===
    retail_current_month REAL DEFAULT 0,         -- 本月零售额
    retail_last_year_month REAL DEFAULT 0,       -- 上年同期零售额

    -- === 分类标记 ===
    is_small_micro INTEGER DEFAULT 0,            -- 小微企业标记
    is_eat_wear_use INTEGER DEFAULT 0,           -- 吃穿用标记

    -- === 补充字段 (输出定稿需要) ===
    first_report_ip TEXT,                        -- 第一次上报的IP
    fill_ip TEXT,                                -- 填报IP
    network_sales REAL DEFAULT 0,                -- 网络销售额
    opening_year INTEGER,                        -- 开业年份
    opening_month INTEGER,                       -- 开业月份

    -- === 原始值备份 ===
    original_revenue_current_month REAL,
    original_room_current_month REAL,
    original_food_current_month REAL,
    original_goods_current_month REAL,

    -- === 元数据 ===
    source_sheet TEXT,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_ac_data_month ON accommodation_catering(data_year, data_month);
CREATE INDEX idx_ac_credit_code ON accommodation_catering(credit_code);
CREATE INDEX idx_ac_industry_type ON accommodation_catering(industry_type);
CREATE INDEX idx_ac_company_scale ON accommodation_catering(company_scale);
```

---

## 3. wr_snapshot - 批零历史快照表

存储历史月份的批零数据快照（如 2024年12月批零、2025年11月批零 等）。

```sql
CREATE TABLE wr_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 快照标识 ===
    snapshot_year INTEGER NOT NULL,              -- 快照年份 (如 2024)
    snapshot_month INTEGER NOT NULL,             -- 快照月份 (如 12)
    snapshot_name TEXT,                          -- 原始 Sheet 名

    -- === 基础信息 ===
    credit_code TEXT,
    name TEXT NOT NULL,
    industry_code TEXT,
    company_scale INTEGER,

    -- === 销售额 ===
    sales_current_month REAL DEFAULT 0,          -- 商品销售额;本年-本月
    sales_current_cumulative REAL DEFAULT 0,     -- 商品销售额;本年-1—本月
    sales_last_year_month REAL,                  -- 商品销售额;上年-本月 (部分快照有)
    sales_last_year_cumulative REAL,             -- 商品销售额;上年-1—本月

    -- === 零售额 ===
    retail_current_month REAL DEFAULT 0,         -- 零售额;本年-本月
    retail_current_cumulative REAL DEFAULT 0,    -- 零售额;本年-1—本月
    retail_last_year_month REAL,                 -- 零售额;上年-本月
    retail_last_year_cumulative REAL,            -- 零售额;上年-1—本月

    -- === 商品分类 ===
    cat_grain_oil_food REAL,
    cat_beverage REAL,
    cat_tobacco_liquor REAL,
    cat_clothing REAL,
    cat_daily_use REAL,
    cat_automobile REAL,

    -- === 元数据 ===
    source_sheet TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wrs_snapshot ON wr_snapshot(snapshot_year, snapshot_month);
CREATE INDEX idx_wrs_credit_code ON wr_snapshot(credit_code);
```

---

## 4. ac_snapshot - 住餐历史快照表

```sql
CREATE TABLE ac_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 快照标识 ===
    snapshot_year INTEGER NOT NULL,
    snapshot_month INTEGER NOT NULL,
    snapshot_name TEXT,

    -- === 基础信息 ===
    credit_code TEXT,
    name TEXT NOT NULL,
    industry_code TEXT,
    company_scale INTEGER,

    -- === 营业额 ===
    revenue_current_month REAL DEFAULT 0,        -- 营业额;本年-本月
    revenue_current_cumulative REAL DEFAULT 0,   -- 营业额;本年-1—本月

    -- === 客房收入 ===
    room_current_month REAL DEFAULT 0,
    room_current_cumulative REAL,

    -- === 餐费收入 ===
    food_current_month REAL DEFAULT 0,
    food_current_cumulative REAL,

    -- === 商品销售额 ===
    goods_current_month REAL DEFAULT 0,
    goods_current_cumulative REAL,

    -- === 元数据 ===
    source_sheet TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_acs_snapshot ON ac_snapshot(snapshot_year, snapshot_month);
CREATE INDEX idx_acs_credit_code ON ac_snapshot(credit_code);
```

---

## 5. sheets_meta - Sheet 元信息表

记录导入的所有 Sheet 信息，支持追溯和扩展。

```sql
CREATE TABLE sheets_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === Sheet 信息 ===
    sheet_name TEXT NOT NULL,                    -- 原始 Sheet 名
    sheet_type TEXT,                             -- 识别类型: wholesale/retail/accommodation/catering/wr_snapshot/ac_snapshot/summary/unknown
    confidence REAL,                             -- 识别置信度 (0-1)

    -- === 统计信息 ===
    total_rows INTEGER DEFAULT 0,                -- 总行数
    total_columns INTEGER DEFAULT 0,             -- 总列数
    imported_rows INTEGER DEFAULT 0,             -- 导入行数

    -- === 列信息 (JSON) ===
    columns_json TEXT,                           -- 原始列名 JSON 数组
    column_mapping_json TEXT,                    -- 字段映射 JSON

    -- === 状态 ===
    status TEXT DEFAULT 'pending',               -- pending/imported/skipped/error
    error_message TEXT,

    -- === 关联 ===
    import_log_id INTEGER,                       -- 关联导入日志

    -- === 元数据 ===
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sm_sheet_type ON sheets_meta(sheet_type);
CREATE INDEX idx_sm_import_log_id ON sheets_meta(import_log_id);
```

---

## 6. config - 系统配置表

```sql
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    value_type TEXT DEFAULT 'string',            -- string/number/json
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 预置配置项
INSERT INTO config (key, value, value_type, description) VALUES
-- 时间配置
('current_year', '2025', 'number', '当前年份'),
('current_month', '12', 'number', '当前月份'),

-- 社零额(定) 手工输入项
('small_micro_rate_month', '0', 'number', '本月小微增速'),
('eat_wear_use_rate_month', '0', 'number', '本月吃穿用增速'),
('sample_rate_month', '0', 'number', '本月抽样单位增速'),
('small_micro_rate_prev', '0', 'number', '上月小微增速'),
('eat_wear_use_rate_prev', '0', 'number', '上月吃穿用增速'),
('sample_rate_prev', '0', 'number', '上月抽样单位增速'),
('weight_small_micro', '0.3', 'number', '小微权重'),
('weight_eat_wear_use', '0.3', 'number', '吃穿用权重'),
('weight_sample', '0.4', 'number', '抽样权重'),
('province_limit_below_rate_change', '0', 'number', '全省限下增速变动量'),

-- 历史累计社零额
('history_social_e18', '0', 'number', '历史累计E18'),
('history_social_e19', '0', 'number', '历史累计E19'),
('history_social_e20', '0', 'number', '历史累计E20'),
('history_social_e21', '0', 'number', '历史累计E21'),
('history_social_e22', '0', 'number', '历史累计E22'),
('history_social_e23', '0', 'number', '历史累计E23'),

-- 汇总表(定) 输入项
('total_company_count', '0', 'number', '单位总数'),
('reported_company_count', '0', 'number', '已上报单位数'),
('negative_growth_count', '0', 'number', '负增长企业数'),

-- 限下社零额
('last_year_limit_below_cumulative', '0', 'number', '上年累计限下社零额');
```

---

## 7. import_logs - 导入日志表

```sql
CREATE TABLE import_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 文件信息 ===
    filename TEXT NOT NULL,
    file_path TEXT,
    file_size INTEGER,
    file_hash TEXT,                              -- MD5 校验

    -- === 导入统计 ===
    total_sheets INTEGER DEFAULT 0,
    imported_sheets INTEGER DEFAULT 0,
    skipped_sheets INTEGER DEFAULT 0,
    total_rows INTEGER DEFAULT 0,
    imported_rows INTEGER DEFAULT 0,
    error_rows INTEGER DEFAULT 0,

    -- === 状态 ===
    status TEXT DEFAULT 'pending',               -- pending/processing/completed/failed
    error_message TEXT,

    -- === 时间 ===
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
```

---

## 字段映射规则

### 批零主表字段映射

| Excel 列名 (模糊匹配) | 数据库字段 |
|----------------------|-----------|
| 统一社会信用代码 | credit_code |
| 单位详细名称 / 企业名称 | name |
| 行业代码 / [201-1] 行业代码 | industry_code |
| 单位规模 | company_scale |
| *年*月销售额 (本年本月) | sales_current_month |
| *年*月商品销售额 (上年同期) | sales_last_year_month |
| *年1-*月销售额 (本年累计) | sales_current_cumulative |
| *年1-*月商品销售额 (上年累计) | sales_last_year_cumulative |
| *年*月零售额 (本年本月) | retail_current_month |
| *年*月商品零售额 (上年同期) | retail_last_year_month |
| 粮油食品类 | cat_grain_oil_food |
| 饮料类 | cat_beverage |
| 烟酒类 | cat_tobacco_liquor |
| 服装鞋帽针纺类 | cat_clothing |
| 日用品类 | cat_daily_use |
| 汽车类 | cat_automobile |
| 小微企业 | is_small_micro |
| 吃穿用 | is_eat_wear_use |

### 住餐主表字段映射

| Excel 列名 (模糊匹配) | 数据库字段 |
|----------------------|-----------|
| *年*月营业额 | revenue_current_month |
| *年*月营业额总计 (上年) | revenue_last_year_month |
| *年*月客房收入 | room_current_month |
| *年*月餐费收入 | food_current_month |
| *年*月销售额 (商品销售额) | goods_current_month |

---

## 行业类型自动判定

```sql
-- 根据行业代码前两位自动设置 industry_type
-- 51: wholesale (批发)
-- 52: retail (零售)
-- 61: accommodation (住宿)
-- 62: catering (餐饮)

-- 触发器示例
CREATE TRIGGER set_wr_industry_type
AFTER INSERT ON wholesale_retail
BEGIN
    UPDATE wholesale_retail SET industry_type =
        CASE
            WHEN substr(NEW.industry_code, 1, 2) = '51' THEN 'wholesale'
            WHEN substr(NEW.industry_code, 1, 2) = '52' THEN 'retail'
            ELSE 'unknown'
        END
    WHERE id = NEW.id AND industry_type IS NULL;
END;

CREATE TRIGGER set_ac_industry_type
AFTER INSERT ON accommodation_catering
BEGIN
    UPDATE accommodation_catering SET industry_type =
        CASE
            WHEN substr(NEW.industry_code, 1, 2) = '61' THEN 'accommodation'
            WHEN substr(NEW.industry_code, 1, 2) = '62' THEN 'catering'
            ELSE 'unknown'
        END
    WHERE id = NEW.id AND industry_type IS NULL;
END;
```

---

## 小微企业自动标记

```sql
-- 单位规模 3 或 4 为小微企业
CREATE TRIGGER set_wr_small_micro
AFTER INSERT ON wholesale_retail
BEGIN
    UPDATE wholesale_retail SET is_small_micro =
        CASE WHEN NEW.company_scale IN (3, 4) THEN 1 ELSE 0 END
    WHERE id = NEW.id;
END;

CREATE TRIGGER set_ac_small_micro
AFTER INSERT ON accommodation_catering
BEGIN
    UPDATE accommodation_catering SET is_small_micro =
        CASE WHEN NEW.company_scale IN (3, 4) THEN 1 ELSE 0 END
    WHERE id = NEW.id;
END;
```

---

## 数据查询示例

### 获取四大行业汇总

```sql
-- 批发业销售额汇总
SELECT
    SUM(sales_current_month) as total_current,
    SUM(sales_last_year_month) as total_last_year,
    SUM(sales_current_cumulative) as total_cumulative,
    SUM(sales_last_year_cumulative) as total_last_cumulative
FROM wholesale_retail
WHERE industry_type = 'wholesale';

-- 零售业零售额汇总
SELECT
    SUM(retail_current_month) as total_current,
    SUM(retail_last_year_month) as total_last_year
FROM wholesale_retail
WHERE industry_type = 'retail';

-- 住宿业营业额汇总
SELECT
    SUM(revenue_current_month) as total_current,
    SUM(revenue_last_year_month) as total_last_year
FROM accommodation_catering
WHERE industry_type = 'accommodation';

-- 餐饮业营业额汇总
SELECT
    SUM(revenue_current_month) as total_current,
    SUM(revenue_last_year_month) as total_last_year
FROM accommodation_catering
WHERE industry_type = 'catering';
```

### 获取小微企业增速

```sql
SELECT
    SUM(retail_current_month) as micro_current,
    SUM(retail_last_year_month) as micro_last_year,
    (SUM(retail_current_month) - SUM(retail_last_year_month)) /
        NULLIF(SUM(retail_last_year_month), 0) * 100 as micro_rate
FROM wholesale_retail
WHERE is_small_micro = 1;
```

### 获取吃穿用增速

```sql
SELECT
    SUM(retail_current_month) as ewu_current,
    SUM(retail_last_year_month) as ewu_last_year,
    (SUM(retail_current_month) - SUM(retail_last_year_month)) /
        NULLIF(SUM(retail_last_year_month), 0) * 100 as ewu_rate
FROM wholesale_retail
WHERE is_eat_wear_use = 1;
```

---

## 月份灵活性设计

### 核心思路

1. **主表存储"当前工作数据"**
   - `wholesale_retail` 和 `accommodation_catering` 表存储当前正在处理的月份数据
   - 通过 `data_year` 和 `data_month` 标识这批数据对应的年月
   - 字段名使用相对时间概念（当月、上月、去年同期等），不硬编码具体月份

2. **快照表存储历史数据**
   - `wr_snapshot` 和 `ac_snapshot` 存储历史月份的快照数据
   - 支持任意月份的历史数据查询和对比

3. **配置表记录当前操作月份**
   - `config` 表中 `current_year` 和 `current_month` 记录当前操作的年月
   - 导入新月份数据时，自动更新配置

### 导入数据流程

```
导入 Excel (如 1月月报.xlsx)
  ↓
1. 解析主表 Sheet (批发/零售/住宿/餐饮)
   - 识别字段中的年月信息 (如 "2026年1月销售额")
   - 提取 data_year=2026, data_month=1
   ↓
2. 清空主表旧数据
   - DELETE FROM wholesale_retail WHERE data_year=2026 AND data_month=1
   - (允许保留其他月份数据用于对比)
   ↓
3. 导入新数据到主表
   - 设置 data_year=2026, data_month=1
   ↓
4. 解析历史快照 Sheet (如 "2025年12月批零")
   - 导入到 wr_snapshot 表
   - 设置 snapshot_year=2025, snapshot_month=12
   ↓
5. 更新配置
   - UPDATE config SET value='2026' WHERE key='current_year'
   - UPDATE config SET value='1' WHERE key='current_month'
```

### 字段语义说明

| 字段名 | 语义 | 示例 (data_month=12) | 示例 (data_month=1) |
|--------|------|---------------------|-------------------|
| `sales_current_month` | 本月销售额 | 2025年12月 | 2026年1月 |
| `sales_prev_month` | 上月销售额 | 2025年11月 | 2025年12月 |
| `sales_last_year_month` | 去年同期 | 2024年12月 | 2025年1月 |
| `sales_current_cumulative` | 本年累计 | 2025年1-12月 | 2026年1月 |
| `sales_prev_cumulative` | 本年累计到上月 | 2025年1-11月 | (无,1月无上月累计) |
| `sales_last_year_cumulative` | 上年累计 | 2024年1-12月 | 2025年1月 |

### 查询当前月份数据

```sql
-- 获取当前操作的年月
SELECT value FROM config WHERE key = 'current_year'; -- 2026
SELECT value FROM config WHERE key = 'current_month'; -- 1

-- 查询当前月份的企业数据
SELECT * FROM wholesale_retail
WHERE data_year = (SELECT value FROM config WHERE key = 'current_year')
  AND data_month = (SELECT value FROM config WHERE key = 'current_month');
```

### 多月份数据对比

```sql
-- 对比 12月 和 1月 的数据
SELECT
    '12月' as month,
    COUNT(*) as company_count,
    SUM(retail_current_month) as total_retail
FROM wholesale_retail
WHERE data_year = 2025 AND data_month = 12

UNION ALL

SELECT
    '1月' as month,
    COUNT(*) as company_count,
    SUM(retail_current_month) as total_retail
FROM wholesale_retail
WHERE data_year = 2026 AND data_month = 1;
```

### 处理特殊情况

1. **1月数据导入**
   - `sales_prev_month`: 上年12月销售额
   - `sales_prev_cumulative`: 不适用（1月无"本年累计到上月"）
   - `sales_current_cumulative`: 等于 `sales_current_month`（1月累计=1月当月）

2. **跨年累计计算**
   - 1月的 `sales_last_year_cumulative` 对应上年1月累计
   - 累计增速计算需要注意分子分母对应同一时间段

3. **Excel 字段动态识别**
   - 解析器需要从列名中提取年月信息
   - 根据年月判断字段映射到数据库的哪个字段
   - 详见 `02_excel_parser.md` 的动态年月识别部分

---

## 补充字段说明

### 输出定稿所需字段

数据库表中包含 5 个补充字段，用于满足输出 "12月月报（定）.xlsx" 的需求：

| 字段名 | 类型 | 用途 | 输出位置 |
|-------|------|------|---------|
| `first_report_ip` | TEXT | 第一次上报的IP | 批零总表、住餐总表 |
| `fill_ip` | TEXT | 填报IP | 批零总表、住餐总表 |
| `network_sales` | REAL | 网络销售额 | 吃穿用 Sheet |
| `opening_year` | INTEGER | 开业年份 | 吃穿用 Sheet |
| `opening_month` | INTEGER | 开业月份 | 吃穿用 Sheet |

### 数据来源策略

#### 方案 A: 从输入 Excel 解析（推荐）

如果输入 Excel 中包含这些字段，解析器自动识别并导入：

```go
// 字段映射规则
var additionalFieldMappings = []MappingRule{
    {Pattern: regexp.MustCompile(`第一次上报.*IP|首次上报IP`), DBField: "first_report_ip"},
    {Pattern: regexp.MustCompile(`填报IP`), DBField: "fill_ip"},
    {Pattern: regexp.MustCompile(`网络销售额`), DBField: "network_sales"},
    {Pattern: regexp.MustCompile(`开业时间.*年|开业年份`), DBField: "opening_year"},
    {Pattern: regexp.MustCompile(`开业时间.*月|开业月份`), DBField: "opening_month"},
}
```

#### 方案 B: 系统记录或用户补充（备选）

如果输入 Excel 中**没有**这些字段：

1. **IP 字段**: 导入时系统自动记录
   ```go
   company.FirstReportIP = ctx.ClientIP()  // 从导入请求获取
   company.FillIP = ""  // 留空
   ```

2. **网络销售额**:
   - 导出时留空或填 0
   - 或提供界面让用户后续补充

3. **开业时间**:
   - 导出时留空
   - 或提供界面让用户后续补充

### 导出处理

导出 "12月月报（定）.xlsx" 时：

```go
// 批零总表/住餐总表
row := []interface{}{
    company.CreditCode,
    company.Name,
    company.IndustryCode,
    // ... 其他字段
    company.FirstReportIP,  // 第16列
    company.FillIP,         // 第17列
}

// 吃穿用 Sheet
row := []interface{}{
    // ... 基础字段
    company.NetworkSales,   // 网络销售额列
    company.OpeningYear,    // 开业年份��
    company.OpeningMonth,   // 开业月份列
}
```

### 字段验证规则

```sql
-- IP 字段格式验证（可选）
CHECK (first_report_ip IS NULL OR first_report_ip LIKE '___.___.___.___')

-- 开业年份范围验证
CHECK (opening_year IS NULL OR (opening_year >= 1900 AND opening_year <= 2100))

-- 开业月份范围验证
CHECK (opening_month IS NULL OR (opening_month >= 1 AND opening_month <= 12))
```

### 实施建议

1. **第一阶段**: 添加字段到表结构，允许 NULL
2. **第二阶段**: 解析器尝试从 Excel 识别并导入
3. **第三阶段**: 导出时处理（有值输出，无值留空）
4. **第四阶段**: 提供数据补充界面（如需要）

### 测试要点

1. 检查输入 Excel 是否包含这 5 个字段
2. 如果包含，验证解析器能否正确识别
3. 如果不包含，验证导出时能否正确留空
4. 验证导出的定稿 Excel 与模板格式完全一致


