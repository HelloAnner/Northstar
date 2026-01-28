# 持久化实现细节（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 目标

- 以最小代价把当前“内存态”（companies/config/import）可靠落到 `data/{projectId}/state.json`
- 自动保存：1000ms debounce 合并写盘；关键节点强制 flush
- 跨平台：macOS/Linux/Windows 都能稳定原子替换（避免 rename 覆盖差异）

## state.json 原子写入（建议）

1. 生成 JSON 内容（包含 `schemaVersion`、时间戳、companies/config/import/indicators）
2. 写入同目录临时文件：`state.json.tmp`
3. `fsync`（可选，当前阶段可不做，保持简单）
4. 覆盖替换：
   - 先尝试 `os.Rename(tmp, state)`（macOS/Linux 通常可覆盖）
   - 若失败且目标文件存在（Windows 常见）：先 `os.Remove(state)` 再 `os.Rename(tmp, state)`
5. 更新 `meta.json.updatedAt` 与 `projects.json.items[*].updatedAt`

## debounce 保存调度（建议）

- 在服务端维护一个 per-project 的 `SaveScheduler`：
  - `ScheduleSave()`：启动/刷新一个 1000ms 定时器
  - `SaveNow()`：取消定时器并立即写盘
- 写盘前应获取互斥锁（避免并发写 `state.json`）

## Excel 落盘时机（需要你确认）

你要的“最新 Excel（latest.xlsx）”有两种语义：

### 方案 A：上传即替换 latest.xlsx

- `POST /import/upload` 成功后就写 `data/{projectId}/latest.xlsx`
- 优点：始终保存用户最后上传的文件
- 缺点：如果用户上传后没执行导入/没配置映射，`latest.xlsx` 可能与 `state.json` 不一致

### 方案 B（推荐）：执行导入成功后才替换 latest.xlsx

- 上传阶段先写临时：`data/{projectId}/uploads/{fileId}.xlsx`（可选）
- `POST /import/:fileId/execute` 成功后，才把该文件复制/rename 成 `latest.xlsx`，同时写入 `state.json`
- 优点：`latest.xlsx` 与 `state.json` 始终一致（同一轮导入的“生效文件”）
- 缺点：需要额外临时文件管理（可在导入成功后清理）

