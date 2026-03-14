# rule-management 模块 E2E-002

> 模块：规则管理
> 版本：002
> 创建：2026-03-14

---

## 一、验收目标

验证 `rules.md -> rules API -> LLM 转换 -> role.json -> engine.ReloadRules()` 的完整闭环已经打通，同时前端 Settings 页面可完成规则管理与状态展示。

## 二、验收场景

### 场景 1：首次启动自动初始化规则文件

前置条件：

- 使用空数据目录启动服务

步骤：

1. 启动服务
2. 检查 `config/rules.md`
3. 调用 `GET /api/rules`
4. 调用 `GET /api/rules/status`

期望：

- 自动创建 `config/rules.md`
- 自动创建 `config/role.json`
- `config/rules.md` 写入内置默认规则清单
- `/api/rules` 返回内置默认规则列表（当前为 17 条）
- `/api/rules/status` 返回 `idle`

### 场景 2：新增规则后自动触发转换

前置条件：

- `rules.md` 已包含默认规则，或由用户清空后再新增
- LLM 返回合法 `role.json`

步骤：

1. 调用 `POST /api/rules`
2. 立即调用 `GET /api/rules/status`
3. 等待转换完成后再次查询状态
4. 读取 `config/role.json`
5. 调用一次 `/api/optimize`

期望：

- `rules.md` 追加新规则并自动编号
- 状态流转为 `running -> ok`
- `role.json` 被写入合法 JSON
- `engine.ReloadRules()` 已生效，`/api/optimize` 返回对应 `appliedRules`

### 场景 3：编辑与删除规则保持文件格式稳定

前置条件：

- `rules.md` 中已有多条规则

步骤：

1. 调用 `PUT /api/rules/:index`
2. 调用 `DELETE /api/rules/:index`
3. 重新读取 `rules.md`
4. 调用 `GET /api/rules`

期望：

- 编辑后的文本被正确覆盖
- 删除后规则重新编号，编号连续
- 文件只保留固定头部与编号列表，不残留旧内容
- 接口返回顺序与文件内容一致

### 场景 4：转换失败时保留旧规则并返回错误详情

前置条件：

- 上一次成功转换后已有有效 `role.json`
- 新一次转换连续 3 轮都未通过校验

步骤：

1. 调用 `POST /api/rules/convert`
2. 轮询 `GET /api/rules/status`
3. 读取 `config/role.json`
4. 调用 `/api/optimize`

期望：

- 状态流转为 `running -> error`
- 返回可读的 `error` 详情
- 旧 `role.json` 不被覆盖
- `/api/optimize` 继续使用旧规则正常工作

### 场景 5：Settings 页面展示规则列表与转换状态

前置条件：

- 前端已进入 `/settings`
- 后端可返回规则列表与状态

步骤：

1. 打开「调整规则」Tab
2. 查看规则列表、状态徽标、最后转换时间
3. 点击新增规则，提交一条规则
4. 观察状态从 `转换中` 到 `已生效`
5. 编辑一条规则，再删除一条规则
6. 当后端返回 `error` 时展开错误详情

期望：

- 列表正确展示编号、文本、编辑、删除入口
- `running` 期间前端自动轮询，`ok/error` 后停止
- Dialog 新增/编辑流程可用
- 错误状态可读且不会阻塞后续再次操作

## 三、通过标准

- 以上 5 个场景全部通过
- 所有新增后端单元测试和前端测试通过
- `go test -tags=e2e` 的 Phase 2 用例通过
- 不破坏 Phase 1 规则引擎能力和既有 Dashboard 主流程

## 四、自动化执行

- 后端闭环验收：`go test -tags=e2e -v ./tests/e2e -run TestRuleManagementPhase2E2E`
- 后端单测：`go test ./internal/rules ./internal/api/v3 ./internal/server`
- 前端测试：`cd web && npm test -- --run`
