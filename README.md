# Northstar

> 月度经济数据统计分析平台，规则驱动 · AI 辅助 · 单文件部署

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=flat-square&logo=react)](https://reactjs.org)
[![SQLite](https://img.shields.io/badge/SQLite-embedded-003B57?style=flat-square&logo=sqlite)](https://sqlite.org)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)

---

## 简介

Northstar 是面向统计工作者的月度经济数据管理工具，管理批发、零售、住宿、餐饮四大行业的月度指标数据。

**核心理念：规则驱动的联动指标调整。** 用户用自然语言定义约束规则，系统自动将规则转换为结构化 JSON，调整引擎在分配过程中实时执行，而非事后校验。

```
自然语言规则 → LLM 转换 → 结构化 RuleSet → 注入 DAG 调整引擎 → 指标实时生效
```

---

## 功能亮点

### 数据管理
- **Excel 导入**：上传 Excel 文件，智能识别列映射，原子性快照替换
- **16 项指标自动计算**：限上社零额、行业增速、社零总额等，DAG 引擎驱动
- **数据导出**：指标与企业明细一键导出为 Excel

### 规则引擎
- **三种规则类型**，覆盖分配全流程：
  - `clamp_target`：分配**前**裁剪目标区间（如：限上增速不超过 15%）
  - `filter_allocation`：分配**中**过滤企业集合（如：仅调整正增长企业）
  - `compensate`：分配**后**自动触发关联补偿（如：批发增速不低于零售）
- **热重载**：规则修改后引擎无停机原子更新
- **降级安全**：规则文件缺失或转换失败时自动回退，不影响正常运行

### AI 对话
- **三种意图**：设定目标值、相对调整、添加持久规则
- **SSE 流式输出**：实时显示 AI 思考与调整结果
- **双层提示词**：内置系统上下文 + 用户可配置偏好层

### 工程特性
- **单文件部署**：前端资源内嵌 Go 二进制，零依赖运行
- **跨平台**：Windows 7/10/11、macOS、Linux
- **离线使用**：无需互联网连接（AI 功能需配置外部 LLM API）

---

## 快速开始

### 直接运行

下载对应平台的二进制文件，直接执行：

```bash
# macOS / Linux
./northstar

# Windows
northstar.exe
```

启动后自动在默认浏览器打开 `http://localhost:20261`

### 自定义配置

在可执行文件同目录下创建 `config.toml`（参考 `config.toml.example`）：

```toml
[server]
port = 20261
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

---

## 架构设计

### 技术栈

| 层次 | 技术 |
|------|------|
| 后端 | Go 1.23 + Gin + SQLite（modernc，纯 Go 编译） |
| 前端 | React 18 + TypeScript + Vite + shadcn/ui + Tailwind CSS |
| AI | LangChainGo，兼容 OpenAI API 格式 |
| 数据处理 | excelize（Excel 读写）|
| 部署 | Go embed 单文件，Makefile 多平台构建 |

### 模块结构

```
northstar/
├── cmd/northstar/          # 主程序入口
├── internal/
│   ├── rules/              # 规则加载、Constraint 类型、LLM 转换
│   │   ├── loader.go       # RuleSet 构建（3 种规则类型）
│   │   ├── constraint.go   # Clamp / Filter / Compensate 执行逻辑
│   │   ├── filter.go       # 4 种企业过滤模式
│   │   └── converter.go    # 异步 LLM 转换（3 轮重试 + 校验）
│   ├── llm/
│   │   ├── system_prompt.go # 内置系统提示词
│   │   ├── intent.go        # 意图解析（set_target / adjust_percent / add_rule）
│   │   ├── tools.go         # LLM Tool 定义
│   │   └── client.go        # LLM 客户端封装
│   ├── dagcalc/
│   │   ├── engine.go        # DAG 引擎，持有 RuleSet，支持热重载
│   │   ├── adjust.go        # 16 指标反向调整算法（三注入点）
│   │   └── adjust_rules.go  # 规则注入桥接层
│   ├── api/v3/              # HTTP API 处理器
│   ├── service/             # Excel 解析、导出、数据存储
│   ├── model/               # 数据模型定义
│   └── config/              # 配置管理
├── web/src/
│   ├── pages/               # DashboardV3、Settings
│   ├── components/          # RuleList、ChatPanel、AIPreferenceForm
│   └── store/               # Zustand 状态管理
├── config/
│   ├── rules.md             # 用户规则（自然语言）
│   └── role.json            # LLM 生成的结构化规则（自动维护）
├── tests/e2e_playwright/    # Playwright E2E 测试
├── docs/                    # 产品与技术设计文档
└── Makefile                 # 构建脚本
```

### 规则引擎数据流

```
用户自然语言 (rules.md)
       ↓  LLM 转换（3 轮重试 + 校验）
结构化规则 (role.json)
       ↓  加载为可执行对象
引擎规则集 (RuleSet)
       ↓  注入 dagcalc 调整算法
调整结果
  ├─ ClampTarget    （分配前：裁剪目标区间）
  ├─ FilterAllocation（分配中：过滤企业集合）
  └─ Compensate     （分配后：关联指标补偿）
```

---

## 16 项指标

| # | 指标 | 类型 |
|---|------|------|
| 1-2 | 限上社零额当月值 / 增速 | 月度 |
| 3-4 | 限上社零额累计值 / 增速 | 累计 |
| 5 | 吃穿用增速（当月） | 分类 |
| 6 | 小微企业增速（当月） | 规模 |
| 7-10 | 批发 / 零售 / 住宿 / 餐饮增速（当月） | 行业 |
| 11-14 | 批发 / 零售 / 住宿 / 餐饮增速（累计） | 行业 |
| 15-16 | 社零总额累计值 / 增速 | 汇总 |

---

## API 参考

### 规则管理

```
GET    /api/rules                # 获取规则列表
POST   /api/rules                # 新增规则
PUT    /api/rules/:index         # 编辑规则
DELETE /api/rules/:index         # 删除规则
POST   /api/rules/convert        # 手动触发 LLM 转换
GET    /api/rules/status         # 查询转换状态
```

### 数据操作

```
GET    /api/v1/indicators        # 查询 16 项指标
GET    /api/v1/companies         # 查询企业列表
PATCH  /api/v1/companies/:id     # 修改企业数据
POST   /api/v1/import/upload     # 上传 Excel 文件
POST   /api/v1/import/:fileId/execute  # 执行数据导入
POST   /api/v1/export            # 导出数据
POST   /api/optimize             # 指标调整
```

### AI 对话

```
POST   /api/llm/chat/stream      # AI 对话（SSE 流式）
GET    /api/settings/user-prompt # 获取用户偏好提示词
PUT    /api/settings/user-prompt # 更新用户偏好提示词
```

---

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
# 分别启动
make start-web       # 终端1：前端开发服务器（Vite HMR）
make start-backend   # 终端2：后端（开发模式）

# 一键启动
make start
```

### 构建

```bash
make build           # 构建当前平台
make build-all       # 构建全部平台（Windows / macOS / Linux）
```

### 测试

```bash
make test                      # Go 单元测试
make test-e2e-pw               # 全部 Playwright E2E 测试
make test-e2e-deterministic    # 排除 LLM 的确定性测试（适合 CI）
make test-e2e-import           # 仅导入导出 + 边缘场景
make test-e2e-ai               # 仅 AI 对话
```

E2E 测试需要配置环境变量以启用 AI 相关用例：

```bash
export LLM_BASE_URL=https://api.openai.com/v1
export LLM_API_KEY=sk-...
export LLM_MODEL_NAME=gpt-4o
```

---

## 许可证

本项目基于 [MIT License](LICENSE) 开源。
