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

    -- === 销售额 (商品销售额) ===
    sales_prev_month REAL DEFAULT 0,             -- 上月销售额 (2025年11月销售额)
    sales_current_month REAL DEFAULT 0,          -- 本月销售额 (2025年12月销售额) ★可调整
    sales_last_year_month REAL DEFAULT 0,        -- 上年同期 (2024年12月商品销售额)
    sales_month_rate REAL,                       -- 12月销售额增速 (计算)
    sales_prev_cumulative REAL DEFAULT 0,        -- 本年1-11月销售额
    sales_last_year_prev_cumulative REAL DEFAULT 0, -- 上年1-11月销售额
    sales_current_cumulative REAL DEFAULT 0,     -- 本年1-12月销售额
    sales_last_year_cumulative REAL DEFAULT 0,   -- 上年1-12月商品销售额
    sales_cumulative_rate REAL,                  -- 1-12月增速 (计算)

    -- === 零售额 ===
    retail_prev_month REAL DEFAULT 0,            -- 上月零售额 (2025年11月零售额)
    retail_current_month REAL DEFAULT 0,         -- 本月零售额 (2025年12月零售额) ★可调整
    retail_last_year_month REAL DEFAULT 0,       -- 上年同期 (2024年12月商品零售额)
    retail_month_rate REAL,                      -- 12月零售额增速 (计算)
    retail_prev_cumulative REAL DEFAULT 0,       -- 本年1-11月零售额
    retail_last_year_prev_cumulative REAL DEFAULT 0, -- 上年1-11月零售额
    retail_current_cumulative REAL DEFAULT 0,    -- 本年1-12月零售额
    retail_last_year_cumulative REAL DEFAULT 0,  -- 上年1-12月商品零售额
    retail_cumulative_rate REAL,                 -- 1-12月增速 (计算)
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

    -- === 营业额 ===
    revenue_prev_month REAL DEFAULT 0,           -- 上月营业额 (2025年11月营业额)
    revenue_current_month REAL DEFAULT 0,        -- 本月营业额 (2025年12月营业额) ★可调整
    revenue_last_year_month REAL DEFAULT 0,      -- 上年同期 (2024年12月营业额)
    revenue_month_rate REAL,                     -- 12月增速 (计算)
    revenue_prev_cumulative REAL DEFAULT 0,      -- 本年1-11月营业额
    revenue_current_cumulative REAL DEFAULT 0,   -- 本年1-12月营业额
    revenue_last_year_cumulative REAL DEFAULT 0, -- 上年1-12月营业额
    revenue_cumulative_rate REAL,                -- 1-12月增速 (计算)

    -- === 客房收入 ===
    room_prev_month REAL DEFAULT 0,              -- 11月客房收入
    room_current_month REAL DEFAULT 0,           -- 本月客房收入 (2025年12月) ★可调整
    room_last_year_month REAL DEFAULT 0,         -- 上年同期客房收入
    room_prev_cumulative REAL DEFAULT 0,         -- 本年1-11月客房收入
    room_current_cumulative REAL DEFAULT 0,      -- 本年1-12月客房收入
    room_last_year_cumulative REAL DEFAULT 0,    -- 上年1-12月客房收入

    -- === 餐费收入 ===
    food_prev_month REAL DEFAULT 0,              -- 11月餐费收入
    food_current_month REAL DEFAULT 0,           -- 本月餐费收入 (2025年12月) ★可调整
    food_last_year_month REAL DEFAULT 0,         -- 上年同期餐费收入
    food_prev_cumulative REAL DEFAULT 0,         -- 本年1-11月餐费收入
    food_current_cumulative REAL DEFAULT 0,      -- 1-12月餐费收入
    food_last_year_cumulative REAL DEFAULT 0,    -- 上年1-12月餐费收入

    -- === 商品销售额 ===
    goods_prev_month REAL DEFAULT 0,             -- 11月销售额
    goods_current_month REAL DEFAULT 0,          -- 本月销售额 (2025年12月) ★可调整
    goods_last_year_month REAL DEFAULT 0,        -- 上年同期商品销售额
    goods_prev_cumulative REAL DEFAULT 0,        -- 本年1-11月销售额
    goods_current_cumulative REAL DEFAULT 0,     -- 1-12月销售额
    goods_last_year_cumulative REAL DEFAULT 0,   -- 上年1-12月商品销售额

    -- === 零售额 (住餐也有) ===
    retail_current_month REAL DEFAULT 0,         -- 本月零售额
    retail_last_year_month REAL DEFAULT 0,       -- 上年同期零售额

    -- === 分类标记 ===
    is_small_micro INTEGER DEFAULT 0,            -- 小微企业标记
    is_eat_wear_use INTEGER DEFAULT 0,           -- 吃穿用标记

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
