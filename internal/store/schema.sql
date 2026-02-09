-- Northstar V3 数据库初始化脚本
-- SQLite 数据库: data/northstar.db
-- 设计文档: specs/003/01_database.md

-- ============================================================================
-- 1. wholesale_retail - 批发零售企业表
-- ============================================================================
CREATE TABLE IF NOT EXISTS wholesale_retail (
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
    sales_prev_month REAL DEFAULT 0,             -- 上月销售额
    sales_current_month REAL DEFAULT 0,          -- 本月销售额 ★可调整
    sales_last_year_month REAL DEFAULT 0,        -- 上年同期
    sales_month_rate REAL,                       -- 当月销售额增速 (计算)
    sales_prev_cumulative REAL DEFAULT 0,        -- 本年累计到上月
    sales_last_year_prev_cumulative REAL DEFAULT 0, -- 上年累计到上月
    sales_current_cumulative REAL DEFAULT 0,     -- 本年累计
    sales_last_year_cumulative REAL DEFAULT 0,   -- 上年累计
    sales_cumulative_rate REAL,                  -- 累计增速 (计算)

    -- === 零售额 ===
    retail_prev_month REAL DEFAULT 0,            -- 上月零售额
    retail_current_month REAL DEFAULT 0,         -- 本月零售额 ★可调整
    retail_last_year_month REAL DEFAULT 0,       -- 上年同期
    零售业销售额增速_当月 REAL,                      -- 当月零售额增速 (计算)
    retail_prev_cumulative REAL DEFAULT 0,       -- 本年累计到上月
    retail_last_year_prev_cumulative REAL DEFAULT 0, -- 上年累计到上月
    retail_current_cumulative REAL DEFAULT 0,    -- 本年累计
    retail_last_year_cumulative REAL DEFAULT 0,  -- 上年累计
    零售业销售额增速_累计 REAL,                 -- 累计增速 (计算)
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
CREATE INDEX IF NOT EXISTS idx_wr_data_month ON wholesale_retail(data_year, data_month);
CREATE INDEX IF NOT EXISTS idx_wr_credit_code ON wholesale_retail(credit_code);
CREATE INDEX IF NOT EXISTS idx_wr_industry_type ON wholesale_retail(industry_type);
CREATE INDEX IF NOT EXISTS idx_wr_company_scale ON wholesale_retail(company_scale);
CREATE INDEX IF NOT EXISTS idx_wr_is_small_micro ON wholesale_retail(is_small_micro);
CREATE INDEX IF NOT EXISTS idx_wr_is_eat_wear_use ON wholesale_retail(is_eat_wear_use);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_wr_key ON wholesale_retail(data_year, data_month, credit_code, industry_type);

-- ============================================================================
-- 2. accommodation_catering - 住宿餐饮企业表
-- ============================================================================
CREATE TABLE IF NOT EXISTS accommodation_catering (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 基础信息 ===
    credit_code TEXT,                            -- 统一社会信用代码
    name TEXT NOT NULL,                          -- 单位详细名称
    industry_code TEXT,                          -- [201-1] 行业代码(GB/T4754-2017)
    industry_type TEXT,                          -- 行业类型: accommodation/catering
    company_scale INTEGER,                       -- 单位规模
    row_no INTEGER,                              -- 原始行号

    -- === 数据月份标识 ===
    data_year INTEGER NOT NULL,                  -- 数据年份
    data_month INTEGER NOT NULL,                 -- 数据月份

    -- === 营业额 ===
    revenue_prev_month REAL DEFAULT 0,           -- 上月营业额
    revenue_current_month REAL DEFAULT 0,        -- 本月营业额 ★可调整
    revenue_last_year_month REAL DEFAULT 0,      -- 上年同期
    revenue_month_rate REAL,                     -- 当月增速 (计算)
    revenue_prev_cumulative REAL DEFAULT 0,      -- 本年累计到上月
    revenue_current_cumulative REAL DEFAULT 0,   -- 本年累计
    revenue_last_year_cumulative REAL DEFAULT 0, -- 上年累计
    revenue_cumulative_rate REAL,                -- 累计增速 (计算)

    -- === 客房收入 ===
    room_prev_month REAL DEFAULT 0,              -- 上月客房收入
    room_current_month REAL DEFAULT 0,           -- 本月客房收入 ★可调整
    room_last_year_month REAL DEFAULT 0,         -- 上年同期客房收入
    room_prev_cumulative REAL DEFAULT 0,         -- 本年累计到上月
    room_current_cumulative REAL DEFAULT 0,      -- 本年累计
    room_last_year_cumulative REAL DEFAULT 0,    -- 上年累计

    -- === 餐费收入 ===
    food_prev_month REAL DEFAULT 0,              -- 上月餐费收入
    food_current_month REAL DEFAULT 0,           -- 本月餐费收入 ★可调整
    food_last_year_month REAL DEFAULT 0,         -- 上年同期餐费收入
    food_prev_cumulative REAL DEFAULT 0,         -- 本年累计到上月
    food_current_cumulative REAL DEFAULT 0,      -- 本年累计
    food_last_year_cumulative REAL DEFAULT 0,    -- 上年累计

    -- === 商品销售额 ===
    goods_prev_month REAL DEFAULT 0,             -- 上月销售额
    goods_current_month REAL DEFAULT 0,          -- 本月销售额 ★可调整
    goods_last_year_month REAL DEFAULT 0,        -- 上年同期商品销售额
    goods_prev_cumulative REAL DEFAULT 0,        -- 本年累计到上月
    goods_current_cumulative REAL DEFAULT 0,     -- 本年累计
    goods_last_year_cumulative REAL DEFAULT 0,   -- 上年累计

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
CREATE INDEX IF NOT EXISTS idx_ac_data_month ON accommodation_catering(data_year, data_month);
CREATE INDEX IF NOT EXISTS idx_ac_credit_code ON accommodation_catering(credit_code);
CREATE INDEX IF NOT EXISTS idx_ac_industry_type ON accommodation_catering(industry_type);
CREATE INDEX IF NOT EXISTS idx_ac_company_scale ON accommodation_catering(company_scale);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_ac_key ON accommodation_catering(data_year, data_month, credit_code, industry_type);

-- ============================================================================
-- 3. wr_snapshot - 批零历史快照表
-- ============================================================================
CREATE TABLE IF NOT EXISTS wr_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === 快照标识 ===
    snapshot_year INTEGER NOT NULL,              -- 快照年份
    snapshot_month INTEGER NOT NULL,             -- 快照月份
    snapshot_name TEXT,                          -- 原始 Sheet 名

    -- === 基础信息 ===
    credit_code TEXT,
    name TEXT NOT NULL,
    industry_code TEXT,
    company_scale INTEGER,

    -- === 销售额 ===
    sales_current_month REAL DEFAULT 0,          -- 商品销售额;本年-本月
    sales_current_cumulative REAL DEFAULT 0,     -- 商品销售额;本年-1—本月
    sales_last_year_month REAL,                  -- 商品销售额;上年-本月
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

CREATE INDEX IF NOT EXISTS idx_wrs_snapshot ON wr_snapshot(snapshot_year, snapshot_month);
CREATE INDEX IF NOT EXISTS idx_wrs_credit_code ON wr_snapshot(credit_code);

-- ============================================================================
-- 4. ac_snapshot - 住餐历史快照表
-- ============================================================================
CREATE TABLE IF NOT EXISTS ac_snapshot (
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

CREATE INDEX IF NOT EXISTS idx_acs_snapshot ON ac_snapshot(snapshot_year, snapshot_month);
CREATE INDEX IF NOT EXISTS idx_acs_credit_code ON ac_snapshot(credit_code);

-- ============================================================================
-- 5. sheets_meta - Sheet 元信息表
-- ============================================================================
CREATE TABLE IF NOT EXISTS sheets_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- === Sheet 信息 ===
    sheet_name TEXT NOT NULL,                    -- 原始 Sheet 名
    sheet_type TEXT,                             -- 识别类型
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

CREATE INDEX IF NOT EXISTS idx_sm_sheet_type ON sheets_meta(sheet_type);
CREATE INDEX IF NOT EXISTS idx_sm_import_log_id ON sheets_meta(import_log_id);

-- ============================================================================
-- 6. sheet_columns - Sheet 列结构
-- ============================================================================
CREATE TABLE IF NOT EXISTS sheet_columns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sheet_name TEXT NOT NULL,
    col_idx INTEGER NOT NULL,                    -- 1-based
    header_text TEXT,                            -- 原始表头文本
    normalized_header TEXT,                      -- 规范化表头
    col_width REAL,                              -- 列宽
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sc_sheet_col ON sheet_columns(sheet_name, col_idx);
CREATE INDEX IF NOT EXISTS idx_sc_import_log_id ON sheet_columns(import_log_id);

-- ============================================================================
-- 7. sheet_rows - Sheet 行结构
-- ============================================================================
CREATE TABLE IF NOT EXISTS sheet_rows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sheet_name TEXT NOT NULL,
    row_idx INTEGER NOT NULL,                    -- 1-based
    row_height REAL,
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sr_sheet_row ON sheet_rows(sheet_name, row_idx);
CREATE INDEX IF NOT EXISTS idx_sr_import_log_id ON sheet_rows(import_log_id);

-- ============================================================================
-- 8. sheet_cells - Sheet 单元格原始数据
-- ============================================================================
CREATE TABLE IF NOT EXISTS sheet_cells (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sheet_name TEXT NOT NULL,
    row_idx INTEGER NOT NULL,
    col_idx INTEGER NOT NULL,
    a1 TEXT,                                     -- A1 坐标
    cell_type TEXT,                              -- 单元格类型
    raw_value TEXT,                              -- 原始值
    formula TEXT,                                -- 公式
    calc_value TEXT,                             -- 计算值（data_only）
    num_format TEXT,                             -- 数字格式
    style_id INTEGER,                            -- 样式 ID
    is_merged INTEGER DEFAULT 0,
    merge_range TEXT,
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sc_cell ON sheet_cells(sheet_name, row_idx, col_idx);
CREATE INDEX IF NOT EXISTS idx_sc_cell_import ON sheet_cells(import_log_id);

-- ============================================================================
-- 9. sheet_merges - Sheet 合并单元格
-- ============================================================================
CREATE TABLE IF NOT EXISTS sheet_merges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sheet_name TEXT NOT NULL,
    merge_range TEXT NOT NULL,                   -- A1:B2
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row INTEGER NOT NULL,
    end_col INTEGER NOT NULL,
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sm_sheet ON sheet_merges(sheet_name);
CREATE INDEX IF NOT EXISTS idx_sm_import_log_id ON sheet_merges(import_log_id);

-- ============================================================================
-- 10. summary_limit_above_retail - 限上零售额汇总
-- ============================================================================
CREATE TABLE IF NOT EXISTS summary_limit_above_retail (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    data_year INTEGER NOT NULL,
    data_month INTEGER NOT NULL,
    row_key TEXT,
    row_no INTEGER,
    value_current REAL,
    value_last REAL,
    rate REAL,
    source_sheet TEXT,
    source_cell TEXT,
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_slar_month ON summary_limit_above_retail(data_year, data_month);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_slar_key ON summary_limit_above_retail(data_year, data_month, row_key);

-- ============================================================================
-- 11. summary_micro_small - 小微汇总
-- ============================================================================
CREATE TABLE IF NOT EXISTS summary_micro_small (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    data_year INTEGER NOT NULL,
    data_month INTEGER NOT NULL,
    row_key TEXT,
    row_no INTEGER,
    value_current REAL,
    value_last REAL,
    rate REAL,
    source_sheet TEXT,
    source_cell TEXT,
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sms_month ON summary_micro_small(data_year, data_month);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_sms_key ON summary_micro_small(data_year, data_month, row_key);

-- ============================================================================
-- 12. summary_eat_wear_use - 吃穿用汇总
-- ============================================================================
CREATE TABLE IF NOT EXISTS summary_eat_wear_use (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    data_year INTEGER NOT NULL,
    data_month INTEGER NOT NULL,
    row_key TEXT,
    row_no INTEGER,
    value_current REAL,
    value_last REAL,
    rate REAL,
    source_sheet TEXT,
    source_cell TEXT,
    import_log_id INTEGER,
    source_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sewu_month ON summary_eat_wear_use(data_year, data_month);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_sewu_key ON summary_eat_wear_use(data_year, data_month, row_key);

-- ============================================================================
-- 13. indicator_definitions - 指标定义
-- ============================================================================
CREATE TABLE IF NOT EXISTS indicator_definitions (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    group_code TEXT NOT NULL,
    group_name TEXT NOT NULL,
    group_order INTEGER DEFAULT 0,
    description TEXT,
    formula TEXT NOT NULL,
    unit TEXT NOT NULL,
    float_min REAL DEFAULT -999,
    float_max REAL DEFAULT 999,
    display_order INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_indicator_group ON indicator_definitions(group_order, display_order);

INSERT OR IGNORE INTO indicator_definitions (
    code, name, group_code, group_name, group_order, description, formula, unit, float_min, float_max, display_order, enabled
) VALUES
    ('限上社零额_当月值', '限上社零额（当月值）', 'limit_above', '限上社零额', 1, '批零零售额 + 住餐折算零售额（当月）',
        'wr_retail_current_month_sum + ac_derived_retail_current_month_sum', '万元', -5, 15, 10, 1),
    ('限上社零额增速_当月', '限上社零额增速（当月）', 'limit_above', '限上社零额', 1, '限上社零额当月同比',
        'percent_diff(限上社零额_当月值, wr_retail_last_year_month_sum + ac_derived_retail_last_year_month_sum)', '%', -10, 20, 20, 1),
    ('限上社零额_累计值', '限上社零额（累计值）', 'limit_above', '限上社零额', 1, '批零零售额 + 住餐折算零售额（累计）',
        'wr_retail_current_cumulative_sum + ac_derived_retail_current_cumulative_sum', '万元', -5, 15, 30, 1),
    ('限上社零额增速_累计', '限上社零额增速（累计）', 'limit_above', '限上社零额', 1, '限上社零额累计同比',
        'percent_diff(限上社零额_累计值, wr_retail_last_year_cumulative_sum + ac_derived_retail_last_year_cumulative_sum)', '%', -10, 20, 40, 1),

    ('吃穿用增速_当月', '吃穿用增速（当月）', 'special_rate', '专项增速', 2, '筛选 is_eat_wear_use=1 后按同比口径计算',
        'percent_diff(wr_eat_wear_use_current_month_sum, wr_eat_wear_use_last_year_month_sum)', '%', -8, 12, 50, 1),
    ('小微企业增速_当月', '小微企业增速（当月）', 'special_rate', '专项增速', 2, '筛选 is_small_micro=1 后按同比口径计算',
        'percent_diff(wr_micro_small_current_month_sum, wr_micro_small_last_year_month_sum)', '%', -5, 18, 60, 1),

    ('批发业销售额增速_当月', '批发业销售额增速（当月）', 'industry_rate', '四大行业增速', 3, '批发行业当月销售额同比',
        'percent_diff(wr_wholesale_sales_current_month_sum, wr_wholesale_sales_last_year_month_sum)', '%', -20, 30, 70, 1),
    ('批发业销售额增速_累计', '批发业销售额增速（累计）', 'industry_rate', '四大行业增速', 3, '批发行业累计销售额同比',
        'percent_diff(wr_wholesale_sales_current_cumulative_sum, wr_wholesale_sales_last_year_cumulative_sum)', '%', -20, 30, 80, 1),
    ('零售业销售额增速_当月', '零售业销售额增速（当月）', 'industry_rate', '四大行业增速', 3, '零售行业当月销售额同比',
        'percent_diff(wr_retail_sales_current_month_sum, wr_retail_sales_last_year_month_sum)', '%', -20, 30, 90, 1),
    ('零售业销售额增速_累计', '零售业销售额增速（累计）', 'industry_rate', '四大行业增速', 3, '零售行业累计销售额同比',
        'percent_diff(wr_retail_sales_current_cumulative_sum, wr_retail_sales_last_year_cumulative_sum)', '%', -20, 30, 100, 1),
    ('住宿业营业额增速_当月', '住宿业营业额增速（当月）', 'industry_rate', '四大行业增速', 3, '住宿行业当月营业额同比',
        'percent_diff(ac_accommodation_revenue_current_month_sum, ac_accommodation_revenue_last_year_month_sum)', '%', -20, 30, 110, 1),
    ('住宿业营业额增速_累计', '住宿业营业额增速（累计）', 'industry_rate', '四大行业增速', 3, '住宿行业累计营业额同比',
        'percent_diff(ac_accommodation_revenue_current_cumulative_sum, ac_accommodation_revenue_last_year_cumulative_sum)', '%', -20, 30, 120, 1),
    ('餐饮业营业额增速_当月', '餐饮业营业额增速（当月）', 'industry_rate', '四大行业增速', 3, '餐饮行业当月营业额同比',
        'percent_diff(ac_catering_revenue_current_month_sum, ac_catering_revenue_last_year_month_sum)', '%', -20, 30, 130, 1),
    ('餐饮业营业额增速_累计', '餐饮业营业额增速（累计）', 'industry_rate', '四大行业增速', 3, '餐饮行业累计营业额同比',
        'percent_diff(ac_catering_revenue_current_cumulative_sum, ac_catering_revenue_last_year_cumulative_sum)', '%', -20, 30, 140, 1),

    ('限下增速_上月', '限下增速（上月）', 'limit_below_model', '限下估算过程指标', 4, '按小微/吃穿用/抽样增速与权重计算上月限下增速',
        'small_micro_rate_prev*weight_small_micro + eat_wear_use_rate_prev*weight_eat_wear_use + sample_rate_prev*weight_sample', '%', -30, 30, 145, 1),
    ('限下增速变动量_本月', '限下增速变动量（本月）', 'limit_below_model', '限下估算过程指标', 4, '本月-上月增速变化叠加全省变动量',
        '(small_micro_rate_month-small_micro_rate_prev)*weight_small_micro + (eat_wear_use_rate_month-eat_wear_use_rate_prev)*weight_eat_wear_use + (sample_rate_month-sample_rate_prev)*weight_sample + province_limit_below_rate_change', '%', -30, 30, 146, 1),
    ('限下增速_本月', '限下增速（本月）', 'limit_below_model', '限下估算过程指标', 4, '本月限下增速 = 上月限下增速 + 本月变动量',
        '限下增速_上月 + 限下增速变动量_本月', '%', -30, 30, 147, 1),
    ('限下累计估算值_累计', '限下累计估算值（累计）', 'limit_below_model', '限下估算过程指标', 4, '本月限下累计 = 上年限下累计 * (1 + 本月限下增速/100)',
        'limit_below_last_cumulative * (1 + 限下增速_本月 / 100)', '万元', -5, 15, 148, 1),
    ('社零总额_累计值', '社零总额（累计值）', 'total_social', '社零总额', 5, '限上累计 + 限下累计估算值',
        '限上社零额_累计值 + 限下累计估算值_累计', '万元', -5, 15, 150, 1),
    ('社零总额增速_累计', '社零总额增速（累计）', 'total_social', '社零总额', 5, '社零总额累计同比',
        'percent_diff(社零总额_累计值, wr_retail_last_year_cumulative_sum + ac_derived_retail_last_year_cumulative_sum + limit_below_last_cumulative)', '%', -10, 20, 160, 1);

-- ============================================================================
-- 14. rule_definitions - 规则定义
-- ============================================================================
CREATE TABLE IF NOT EXISTS rule_definitions (
    rule_code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    expression TEXT,
    severity TEXT DEFAULT 'warn',
    suggestion TEXT,
    preference_json TEXT DEFAULT '{}',
    display_order INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rule_order ON rule_definitions(display_order);

INSERT OR IGNORE INTO rule_definitions (
    rule_code, name, description, expression, severity, suggestion, preference_json, display_order, enabled
) VALUES
    ('rule_phase2_industry_growth_limit', '规则P2-1 行业增速区间与差异约束', '四大行业当月/累计增速控制在±30%，企业间差异控制在1%浮动区间',
        'abs(industry_month_rate) <= rule_growth_abs_limit && abs(industry_cumulative_rate) <= rule_growth_abs_limit', 'warn',
        '超限时优先回调行业样本企业并控制增速离散度', '{"abs_limit":30,"jitter_limit":1}', 90, 1),
    ('rule_phase2_wholesale_ratio_limit', '规则P2-2 批发业零销比约束', '批发业零销比一般不超过40%，大个体不超过30%，且小数位不雷同',
        'wholesale_retail_ratio <= rule_wholesale_ratio_limit', 'warn',
        '重点检查大个体企业零销比与零售额占比', '{"ratio_limit":0.4,"big_ratio_limit":0.3,"unique_decimal":true}', 100, 1),
    ('rule_phase2_retail_ratio_limit', '规则P2-3 零售业零销比约束', '乡镇加油站零销比约50%，其他企业销售额约等于零售额，大个体增速不超过20%',
        'retail_growth_rate <= rule_retail_big_growth_limit', 'warn',
        '异常时优先校验乡镇加油站与大个体企业', '{"gas_station_ratio_target":0.5,"big_growth_limit":20}', 110, 1),
    ('rule_phase2_accommodation_room_food_relation', '规则P2-4 住宿业客房与餐费关系', '有餐饮的住宿企业客房收入需高于餐费收入',
        '(has_food == 0) || (room_income > food_income)', 'warn',
        '优先调节客房收入并保持营业额口径一致', '{"relation":"room_gt_food"}', 120, 1),
    ('rule_phase2_catering_room_food_relation', '规则P2-5 餐饮业客房与餐费关系', '有住宿的餐饮企业客房收入需低于餐费收入，纯住餐营业额需等于客房或餐费',
        '(has_room == 0) || (room_income < food_income)', 'warn',
        '出现冲突时按企业类型分别校准客房与餐费字段', '{"relation":"room_lt_food","pure_business_equal":true}', 130, 1),
    ('rule_phase2_hotel_catering_decimal_stability', '规则P2-6 住餐增速小数稳定约束', '住宿/餐饮销售额与零售额增速上下浮动个位保持不变，小数后一位变化≤1%',
        'abs(rate_decimal_delta) <= rule_room_food_delta_limit', 'warn',
        '优先进行微调，避免打破原有小数分布', '{"decimal_delta_limit":1}', 140, 1),
    ('rule_phase2_big_individual_limit', '规则P2-7 大个体增速约束', '大个体增速不超过法人增速；乡镇超市当月≤100万；当月增速≤20%',
        'big_individual_growth_rate <= 20', 'warn',
        '对大个体企业采用更保守的分配系数', '{"max_month_value":1000000,"max_growth":20}', 150, 1),
    ('rule_phase2_wholesale_decimal_stability', '规则P2-8 批发业增速小数稳定约束', '批发业销售额、零售额增速上下浮动个位保持不变，小数后一位变化≤1%',
        'abs(wholesale_rate_decimal_delta) <= 1', 'warn',
        '优先调整批发行业头部企业的本期值', '{"decimal_delta_limit":1}', 160, 1),
    ('rule_phase2_new_company_caps', '规则P2-9 新进企业同期累计上限', '新进企业同期累计上限：批发≤2000万、零售≤500万、住餐≤200万',
        'new_company_cap_check == 1', 'warn',
        '新进企业数据异常时先校验同期累计基数', '{"wholesale_cap":2000,"retail_cap":500,"accommodation_catering_cap":200}', 170, 1),
    ('rule_phase2_priority_maximize', '规则P2-10 小微与吃穿用优先策略', '小微与吃穿用增速优先最大化，目标靠近30%',
        'priority_target >= rule_priority_target', 'info',
        '智能调整时优先分配到小微与吃穿用相关企业', '{"priority_target":30,"priority_items":["小微企业增速_当月","吃穿用增速_当月"]}', 180, 1);

-- ============================================================================
-- 15. rule_indicator_links - 规则与指标联动关系
-- ============================================================================
CREATE TABLE IF NOT EXISTS rule_indicator_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_code TEXT NOT NULL,
    indicator_code TEXT NOT NULL,
    relation_label TEXT,
    weight REAL DEFAULT 0,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rule_code, indicator_code)
);

CREATE INDEX IF NOT EXISTS idx_rule_indicator_links_rule ON rule_indicator_links(rule_code, display_order);
CREATE INDEX IF NOT EXISTS idx_rule_indicator_links_indicator ON rule_indicator_links(indicator_code);

INSERT OR IGNORE INTO rule_indicator_links (rule_code, indicator_code, relation_label, weight, display_order) VALUES
    ('rule_phase2_industry_growth_limit', '批发业销售额增速_当月', '批发当月增速区间约束', 0.16, 60),
    ('rule_phase2_industry_growth_limit', '批发业销售额增速_累计', '批发累计增速区间约束', 0.16, 70),
    ('rule_phase2_industry_growth_limit', '零售业销售额增速_当月', '零售当月增速区间约束', 0.17, 80),
    ('rule_phase2_industry_growth_limit', '零售业销售额增速_累计', '零售累计增速区间约束', 0.17, 90),
    ('rule_phase2_industry_growth_limit', '住宿业营业额增速_当月', '住宿当月增速区间约束', 0.17, 100),
    ('rule_phase2_industry_growth_limit', '住宿业营业额增速_累计', '住宿累计增速区间约束', 0.17, 110),
    ('rule_phase2_industry_growth_limit', '餐饮业营业额增速_当月', '餐饮当月增速区间约束', 0.17, 120),
    ('rule_phase2_industry_growth_limit', '餐饮业营业额增速_累计', '餐饮累计增速区间约束', 0.17, 130),

    ('rule_phase2_wholesale_ratio_limit', '批发业销售额增速_当月', '批发零销比联动约束', 1, 140),
    ('rule_phase2_retail_ratio_limit', '零售业销售额增速_当月', '零售零销比联动约束', 1, 150),
    ('rule_phase2_accommodation_room_food_relation', '住宿业营业额增速_当月', '住宿客房/餐费约束', 1, 160),
    ('rule_phase2_catering_room_food_relation', '餐饮业营业额增速_当月', '餐饮客房/餐费约束', 1, 170),
    ('rule_phase2_hotel_catering_decimal_stability', '住宿业营业额增速_当月', '住宿增速小数稳定', 0.5, 180),
    ('rule_phase2_hotel_catering_decimal_stability', '餐饮业营业额增速_当月', '餐饮增速小数稳定', 0.5, 190),
    ('rule_phase2_big_individual_limit', '小微企业增速_当月', '大个体与小微增速约束', 1, 200),
    ('rule_phase2_wholesale_decimal_stability', '批发业销售额增速_当月', '批发增速小数稳定', 1, 210),
    ('rule_phase2_new_company_caps', '限上社零额_当月值', '新进企业累计上限影响限上值', 1, 220),
    ('rule_phase2_priority_maximize', '小微企业增速_当月', '小微优先最大化目标', 0.5, 230),
    ('rule_phase2_priority_maximize', '吃穿用增速_当月', '吃穿用优先最大化目标', 0.5, 240);


DELETE FROM rule_indicator_links WHERE rule_code IN (
    'rule_limit_above_sum',
    'rule_rate_formula',
    'rule_limit_below_08_1',
    'rule_limit_below_08_2',
    'rule_limit_below_08_3',
    'rule_limit_below_08_4',
    'rule_total_social',
    'rule_optimize_notice'
);

DELETE FROM rule_definitions WHERE rule_code IN (
    'rule_limit_above_sum',
    'rule_rate_formula',
    'rule_limit_below_08_1',
    'rule_limit_below_08_2',
    'rule_limit_below_08_3',
    'rule_limit_below_08_4',
    'rule_total_social',
    'rule_optimize_notice'
);

UPDATE rule_definitions SET expression = '(has_food == 0) || (room_income > food_income)' WHERE rule_code = 'rule_phase2_accommodation_room_food_relation';
UPDATE rule_definitions SET expression = '(has_room == 0) || (room_income < food_income)' WHERE rule_code = 'rule_phase2_catering_room_food_relation';
UPDATE rule_definitions SET expression = 'new_company_cap_check == 1' WHERE rule_code = 'rule_phase2_new_company_caps';
UPDATE rule_definitions SET expression = 'priority_target >= rule_priority_target' WHERE rule_code = 'rule_phase2_priority_maximize';

UPDATE indicator_definitions SET formula = '限上社零额_累计值 + 限下累计估算值_累计', group_order = 5 WHERE code = '社零总额_累计值';
UPDATE indicator_definitions SET group_order = 5 WHERE code = '社零总额增速_累计';

-- ============================================================================
-- 16. config - 系统配置表
-- ============================================================================
CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    value_type TEXT DEFAULT 'string',            -- string/number/json
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 预置配置项
INSERT OR IGNORE INTO config (key, value, value_type, description) VALUES
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
('last_year_limit_below_cumulative', '0', 'number', '上年累计限下社零额'),

-- 规则默认阈值
('rule_growth_abs_limit', '30', 'number', '行业增速绝对值上限（%）'),
('rule_growth_jitter_limit', '1', 'number', '行业增速离散浮动阈值（%）'),
('rule_wholesale_ratio_limit', '0.4', 'number', '批发业零销比上限'),
('rule_wholesale_big_ratio_limit', '0.3', 'number', '批发业大个体零销比上限'),
('rule_retail_gas_station_ratio_target', '0.5', 'number', '乡镇加油站零销比目标'),
('rule_retail_big_growth_limit', '20', 'number', '零售业大个体增速上限（%）'),
('rule_room_food_delta_limit', '1', 'number', '住餐增速小数位变化阈值（%）'),
('rule_new_company_wholesale_year_cap', '2000', 'number', '新进企业批发累计上限（万）'),
('rule_new_company_retail_year_cap', '500', 'number', '新进企业零售累计上限（万）'),
('rule_new_company_ac_year_cap', '200', 'number', '新进企业住餐累计上限（万）'),
('rule_priority_target', '30', 'number', '小微与吃穿用优先目标增速（%）'),

-- 大模型配置
('llm_base_url', '', 'string', '大模型 Base URL'),
('llm_model', '', 'string', '大模型模型'),
('llm_api_key', '', 'string', '大模型 API Key');

-- ============================================================================
-- 7. import_logs - 导入日志表
-- ============================================================================
CREATE TABLE IF NOT EXISTS import_logs (
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

-- ============================================================================
-- 触发器 - 自动设置行业类型
-- ============================================================================
CREATE TRIGGER IF NOT EXISTS set_wr_industry_type
AFTER INSERT ON wholesale_retail
FOR EACH ROW
WHEN NEW.industry_type IS NULL AND NEW.industry_code IS NOT NULL
BEGIN
    UPDATE wholesale_retail SET industry_type =
        CASE
            WHEN substr(NEW.industry_code, 1, 2) = '51' THEN 'wholesale'
            WHEN substr(NEW.industry_code, 1, 2) = '52' THEN 'retail'
            ELSE 'unknown'
        END
    WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS set_ac_industry_type
AFTER INSERT ON accommodation_catering
FOR EACH ROW
WHEN NEW.industry_type IS NULL AND NEW.industry_code IS NOT NULL
BEGIN
    UPDATE accommodation_catering SET industry_type =
        CASE
            WHEN substr(NEW.industry_code, 1, 2) = '61' THEN 'accommodation'
            WHEN substr(NEW.industry_code, 1, 2) = '62' THEN 'catering'
            ELSE 'unknown'
        END
    WHERE id = NEW.id;
END;

-- ============================================================================
-- 触发器 - 自动标记小微企业
-- ============================================================================
CREATE TRIGGER IF NOT EXISTS set_wr_small_micro
AFTER INSERT ON wholesale_retail
FOR EACH ROW
WHEN NEW.company_scale IN (3, 4)
BEGIN
    UPDATE wholesale_retail SET is_small_micro = 1
    WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS set_ac_small_micro
AFTER INSERT ON accommodation_catering
FOR EACH ROW
WHEN NEW.company_scale IN (3, 4)
BEGIN
    UPDATE accommodation_catering SET is_small_micro = 1
    WHERE id = NEW.id;
END;

-- ============================================================================
-- 触发器 - 更新时间戳
-- ============================================================================
CREATE TRIGGER IF NOT EXISTS update_wr_timestamp
AFTER UPDATE ON wholesale_retail
FOR EACH ROW
BEGIN
    UPDATE wholesale_retail SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_ac_timestamp
AFTER UPDATE ON accommodation_catering
FOR EACH ROW
BEGIN
    UPDATE accommodation_catering SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_config_timestamp
AFTER UPDATE ON config
FOR EACH ROW
BEGIN
    UPDATE config SET updated_at = CURRENT_TIMESTAMP
    WHERE key = NEW.key;
END;

CREATE TRIGGER IF NOT EXISTS update_indicator_definitions_timestamp
AFTER UPDATE ON indicator_definitions
FOR EACH ROW
BEGIN
    UPDATE indicator_definitions SET updated_at = CURRENT_TIMESTAMP
    WHERE code = NEW.code;
END;

CREATE TRIGGER IF NOT EXISTS update_rule_definitions_timestamp
AFTER UPDATE ON rule_definitions
FOR EACH ROW
BEGIN
    UPDATE rule_definitions SET updated_at = CURRENT_TIMESTAMP
    WHERE rule_code = NEW.rule_code;
END;
