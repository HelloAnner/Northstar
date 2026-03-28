# NorthStar 项目指南

## 文档体系

### 全局设计文档 `docs/`

| 文件 | 作用 | 参考权重 |
|------|------|----------|
| `prd.md` | 产品设计，需求的来源 | 高 |
| `tech.md` | 技术架构设计 | **最高** |

### 业务模块文档 `docs/<模块名>/`

每个子目录是一个业务模块，包含一个 `spec.md` 描述该模块的最新现状：

```
docs/
  prd.md                  # 全局产品设计
  tech.md                 # 全局技术架构
  rule-engine/spec.md     # 规则引擎模块
  rule-management/spec.md # 规则管理模块
  ai-chat/spec.md         # AI 对话模块
```

---

## 开发流程

实现一个业务模块功能前，**必须先完成文档**，再写代码：

```
1. 阅读 docs/tech.md 了解全局架构
2. 阅读或更新目标模块的 docs/<模块>/spec.md
3. 实现前后端代码
4. 实现完成后，更新 spec.md 反映最新现状
```

---

## E2E 测试规范

### 测试框架

- 目录：`tests/e2e_playwright/`
- 框架：Playwright for Python + pytest
- 报告：自定义 HTML 报告（`report_plugin.py`），自包含 Base64 截图
- 运行：`make test-e2e-pw`（全部）/ `make test-e2e-deterministic`（排除 LLM）

### 测试分类（6 个文件，70 个用例）

| 文件 | 用例数 | 说明 |
|------|--------|------|
| `test_01_import_export.py` | 10 | 导入导出功能（Excel 上传、SSE 事件、Sheet 识别、导出验证） |
| `test_02_dashboard.py` | 8 | 仪表盘交互（指标卡片、企业表格、搜索、Tab 切换） |
| `test_03_rules.py` | 10 | 规则管理（CRUD、转换、生效验证、UI 操作） |
| `test_04_ai_chat.py` | 12 | AI 对话（真实 LLM，`@pytest.mark.llm`，3 种意图、多轮对话） |
| `test_05_data_consistency.py` | 7 | 数据一致性（roundtrip、指标一致、累计值、行业汇总） |
| `test_06_excel_edge_cases.py` | 25 | Excel 边缘场景（Sheet/列/数据/日期/文件 25 种变体） |

### 架构要点

- **双通道**：数据准备走 API（`helpers/api_client.py`），UI 验证走 Playwright
- **服务器自管理**：`helpers/server.py` 自动 `go build` → 启动 → 配置 LLM → 测试 → 停止
- **LLM 配置**：读取环境变量 `LLM_BASE_URL` / `LLM_API_KEY` / `LLM_MODEL_NAME`，无配置时 LLM 测试自动 skip
- **截图**：每个使用 `page` fixture 的测试自动截图，报告中 Base64 内嵌
- **Edge Case**：`helpers/excel_factory.py` 从 PRD Excel 动态生成变体

### 添加新测试

1. 在对应文件中添加测试方法，中文 docstring 作为报告描述
2. 需要真实 LLM 的测试标记 `@pytest.mark.llm`
3. 需要浏览器交互的测试使用 `page` fixture
4. 纯 API 测试使用 `api` fixture
5. 依赖已导入数据的测试使用 `import_data` fixture

### Makefile 命令

```bash
make e2e-deps              # 安装依赖（playwright + python 包）
make test-e2e-pw           # 全部 Playwright E2E
make test-e2e-deterministic # 排除 LLM 测试（适合 CI）
make test-e2e-import       # 仅导入导出 + 边缘场景
make test-e2e-ai           # 仅 AI 对话
```

---
