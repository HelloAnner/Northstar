# Northstar - 经济数据统计分析工具

社零额数据统计分析与模拟平台，支持 Excel 数据导入、16 项指标实时计算、智能调整等功能。

## 功能特点

- **单文件部署**: 所有前端资源内嵌到 Go 二进制文件
- **自动打开浏览器**: 启动后自动在默认浏览器打开工具
- **离线使用**: 无需互联网连接即可正常运行
- **跨平台支持**: Windows 7/10/11、macOS、Linux

## 快速开始

### 直接运行

```bash
# macOS/Linux
./northstar

# Windows
northstar.exe
```

启动后会自动在浏览器打开 `http://localhost:8080`

### 自定义配置

在可执行文件同目录下创建 `config.toml`:

```toml
[server]
port = 8080
dev_mode = false

[data]
data_dir = "data"
auto_backup = true

[business]
default_month = 1
max_growth = 0.5
min_growth = -0.3
```

### 命令行参数

```bash
./northstar -port 9000    # 指定端口
./northstar -dev          # 开发模式（不自动打开浏览器）
```

## 开发

### 环境要求

- Go 1.21+
- Node.js 18+
- Make

### 安装依赖

```bash
make deps
```

### 开发模式启动

```bash
# 终端1: 启动前端开发服务器
make start-web

# 终端2: 启动后端（开发模式）
make start-backend
```

或一键启动:

```bash
make start
```

### 构建

```bash
# 构建当前平台
make build

# 构建全部平台 (Windows/macOS/Linux)
make build-all
```

### 测试

```bash
# Go 单元测试
make test

# 端到端测试 (需要先构建)
make auto-test
```

## 项目结构

```
Northstar/
├── cmd/northstar/          # 主程序入口
├── internal/
│   ├── config/             # 配置管理
│   ├── model/              # 数据模型
│   ├── server/             # HTTP 服务器
│   │   ├── handlers/       # API 处理器
│   │   └── dist/           # 嵌入的前端资源
│   ├── service/
│   │   ├── calculator/     # 指标计算引擎
│   │   ├── excel/          # Excel 解析/导出
│   │   └── store/          # 内存数据存储
│   └── util/               # 工具函数
├── web/                    # 前端项目
│   ├── src/
│   │   ├── pages/          # 页面组件
│   │   ├── services/       # API 服务
│   │   ├── store/          # 状态管理
│   │   └── types/          # TypeScript 类型
│   └── ...
├── tests/e2e/              # E2E 测试
├── specs/                  # 设计文档
├── Makefile                # 构建脚本
└── config.toml.example     # 配置示例
```

## API 接口

### 指标查询

```
GET /api/v1/indicators
```

### 企业管理

```
GET    /api/v1/companies
GET    /api/v1/companies/:id
PATCH  /api/v1/companies/:id
POST   /api/v1/companies/reset
```

### 数据导入

```
POST   /api/v1/import/upload
GET    /api/v1/import/:fileId/columns
POST   /api/v1/import/:fileId/mapping
POST   /api/v1/import/:fileId/execute
```

### 智能调整

```
POST   /api/v1/optimize
POST   /api/v1/optimize/preview
```

### 数据导出

```
POST   /api/v1/export
GET    /api/v1/export/download/:exportId
```

## 16 项指标

1. 限上社零额(当月值)
2. 限上社零额增速(当月)
3. 限上社零额(累计值)
4. 限上社零额增速(累计)
5. 吃穿用增速(当月)
6. 小微企业增速(当月)
7-14. 四大行业销售额增速(当月/累计)
15. 社零总额(累计值)
16. 社零总额增速(累计)

## 指标与数据关系 DAG（ASCII）

```
                         +------------------------------+
                         |         配置表 Config         |
                         | lastYearLimitBelowCumulative |
                         +---------------+--------------+
                                         |
                                         v
                              +----------+-----------+
                              |  估算限下社零额(累计)   |
                              |  = lastYearLimitBelow  |
                              |    * (1 + 小微增速)     |
                              +----------+-----------+
                                         |
                                         v
+------------------------------+    +----+---------------------------+
|          企业表 Company       |    | 社零总额(累计值)               |
|  - RetailCurrentMonth         |    | = 限上累计 + 估算限下累计       |
|  - RetailLastYearMonth        |    +----+---------------------------+
|  - RetailCurrentCumulative    |         |
|  - RetailLastYearCumulative   |         v
|  - SalesCurrentMonth          |    +----+---------------------------+
|  - SalesLastYearMonth         |    | 社零总额增速(累计)             |
|  - SalesCurrentCumulative     |    | = (本年社零总额 - 上年社零总额) |
|  - SalesLastYearCumulative    |    |   / 上年社零总额               |
|  - IndustryType               |    +--------------------------------+
|  - IsEatWearUse               |
|  - CompanyScale(3/4为小微)     |
+---------------+--------------+
                |
                v
      +---------+-------------------------------+
      |           预聚合 Sums                   |
      |  AllRetailCurrent / AllRetailLastYear   |
      |  AllRetailCurrentCumulative / LastYear  |
      |  EatWearUseRetailCurrent / LastYear     |
      |  MicroSmallRetailCurrent / LastYear     |
      |  IndustrySales(4行业: 批发/零售/住宿/餐饮)|
      +---------+-------------------------------+
                |
                +------------------------------+
                |                              |
                v                              v
   +------------+------------------+   +-------+----------------------+
   | 限上社零额(当月值/累计值)     |   | 四大行业增速(当月/累计)       |
   | = Σ零售额                     |   | = Σ销售额增速(行业维度)        |
   +------------+------------------+   +------------------------------+
                |
                v
   +------------+------------------+
   | 限上社零额增速(当月/累计)     |
   | = (本期 - 上年同期) / 上年同期|
   +------------------------------+

   +------------+------------------+
   | 吃穿用增速(当月)               |
   | = 吃穿用零售额增速             |
   +------------------------------+

   +------------+------------------+
   | 小微企业增速(当月)             |
   | = 小微零售额增速               |
   +------------------------------+
```

## 许可证

MIT
