# FR-012 自举验收指南

本文用于在获批的隔离 PingCode 测试项目中验证一次创建、读取、更新和回读闭环。所有真实租户操作都属于 `INTEGRATION`；mock 或协议测试不能替代该验收。关联 PRD：FR-012；OpenSpec：`bootstrap-pingcode-mcp` task 5.3；PingCode 需求 ID：`PENDING`。

## 前置授权

以下条件全部满足后才能开始真实自举：

1. 负责人提供可用的 Client ID 和 Client Secret。只保存在本地进程环境或仓库忽略的 `.pingcode.env` 中，不提交、不粘贴到对话记录；启动进程时将值加载为环境变量。
2. 负责人指定一个隔离测试项目，并提供其项目 ID。不得使用生产项目。
3. 负责人明确授权在该隔离项目创建和更新测试工作项。
4. 自举操作期间才允许设置 `PINGCODE_READ_ONLY=false`；`PINGCODE_ALLOWED_PROJECT_IDS` 只加入获批的隔离项目 ID。

任一条件缺失时只可进行离线 mock/协议测试，不得向真实 PingCode 发起写入。

## 验收步骤

下列 JSON 是 MCP 工具参数形态示例。将 `<approved-project-id>`、`<approved-type-id>` 等占位符替换为负责人提供且已核对的值；不要把凭据放入工具参数。每一步记录实际结果摘要。

### 1. 只读预检

先调用 `pingcode_list_projects`，确认返回的项目都属于配置的白名单；如果目标项目在当前页未出现，使用 `page_index` 检查其他页。该工具在每个 PingCode 分页结果上本地过滤白名单，`total` 表示当前页过滤后数量。

```json
{
  "page_size": 100,
  "page_index": 0
}
```

再用 `pingcode_get_project` 验证目标项目 ID：

```json
{
  "project_id": "<approved-project-id>"
}
```

预期：项目详情成功返回，目标 `project_id` 在白名单中。若工具报告 `PROJECT_NOT_ALLOWED`，停止操作并核对本地白名单。

### 2. 创建单个工作项

调用 `pingcode_create_work_item`，只提供所需的最小字段集。`type_id` 必须来自该项目可用且已由负责人确认的工作项类型。

```json
{
  "project_id": "<approved-project-id>",
  "type_id": "<approved-type-id>",
  "title": "MCP bootstrap check YYYY-MM-DD"
}
```

预期：返回新工作项，记录其 `id` 供后续步骤使用。若收到 `READ_ONLY` 或其他错误，不要反复提交创建请求；先按错误处置建议核对配置与授权。

### 3. 回读创建结果

调用 `pingcode_get_work_item`：

```json
{
  "workitem_id": "<created-work-item-id>",
  "project_id": "<approved-project-id>"
}
```

预期：回读的工作项 `id`、`title`、`type` 和 `project` 与创建目标一致。

### 4. 更新工作项

选择更新标题或描述中的一项，调用 `pingcode_update_work_item`。示例只更新标题：

```json
{
  "workitem_id": "<created-work-item-id>",
  "project_id": "<approved-project-id>",
  "title": "MCP bootstrap check YYYY-MM-DD updated"
}
```

预期：更新调用成功。工具只发送本次提供的更新字段；不要在本步骤改动其他项目数据。

### 5. 终验回读

再次按步骤 3 的参数调用 `pingcode_get_work_item`。

预期：`id`、`project` 仍指向隔离测试项目，且 `title`（或所选的 `description`）已显示更新后的值。

### 6. 立即恢复只读

停止当前自举进程或修改其本地环境，将 `PINGCODE_READ_ONLY` 恢复为 `true`，并重新启动服务。确认后续写操作会被拒绝。不得让自举配置继续用于日常会话。

## 证据记录

在负责人认可的位置记录以下信息：

- 操作日期和时间。
- 目标隔离项目 ID；可按负责人要求脱敏。
- 创建得到的工作项 ID。
- 六个步骤各自的结果摘要，以及终验回读的关键字段是否符合预期。
- 本次操作的授权范围和 `PINGCODE_READ_ONLY` 恢复结果。

不得记录 Client Secret、Token、完整请求 URL 或敏感工作项正文。若观察到的租户行为与 [docs/api-contract.md](api-contract.md) 不一致，记录脱敏差异和复现步骤，并在契约文档中按维护流程跟进；不要把尚未确认的行为写成已验证契约。

## 回退

1. 立即将 `PINGCODE_READ_ONLY` 恢复为 `true`，必要时停止自举 MCP 进程并以只读环境重新启动。
2. 如需清理测试工作项，通过 PingCode Web 按负责人批准的范围处理；P0 MCP 不提供删除工具。
3. 如怀疑凭据泄露，通知负责人并轮换 Client Secret；不要在日志或对话中粘贴旧值或新值。

## 验收记录（INTEGRATION-PENDING）

本公开版本不包含任何真实租户、项目、工作项标识符或操作时间。真实自举必须由维护者在获批的隔离测试项目中按本指南执行，并将证据保存在不进入公开仓库的受控位置。mock 和 MCP 协议测试不能替代真实租户验收。

公开仓库中仅保留以下可复现的验收要求：

- 先完成只读预检，再执行单项创建、回读、更新和终验回读。
- 真实写入必须明确标记为 `INTEGRATION`，使用获批项目白名单，并临时设置 `PINGCODE_READ_ONLY=false`。
- 验收记录应在私有位置保存脱敏的授权范围、操作时间、资源标识符和结果摘要；不得复制到公开文档。
- 完成后恢复 `PINGCODE_READ_ONLY=true`，并验证后续写操作会被本地拒绝。

## 当前状态

FR-012 在本公开文档中保持 `INTEGRATION-PENDING`。公开的 mock、HTTP mock 和 MCP 协议测试不构成真实租户联调证据；取得独立授权后，维护者可按本指南完成并私下记录验收。
