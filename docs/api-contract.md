# PingCode API 契约（公开发布摘要 / FR-001）

> 本文件只记录供应商公开 API 的泛化契约。公开仓库不保存内部租户、项目、工作项、操作时间、运行探针、内部链接、本机路径或凭据。

## 1. 元信息

| 项目 | 值 |
|---|---|
| PRD 追踪 | `FR-001`（官方 API 研究与契约验证） |
| 公开来源 | `https://open.pingcode.com/api_data.js` |
| 发布材料 | `.kiro/reference/pingcode-api_data.js`、`.kiro/reference/pingcode-api-extract-p0.md` |
| 数据状态 | `public-contract-only`；仅保留公开 endpoint、参数、scope 和 schema |
| 验证标记 | `DOC-VERIFIED` 表示供应商公开文档中有对应契约；`INTEGRATION-PENDING` 表示需要获准的真实环境确认 |

两份 reference 文件不是运行导出，也不代表任何租户已授权或已联调。契约摘要中的 ID、时间字段和路径参数均为 schema 定义，不提供实际值。

## 2. 环境与鉴权

### 2.1 URL 根路径

- 公有云 REST 根：`https://open.pingcode.com`，资源路径使用 `/v1/...`。
- 私有部署根路径由部署方配置；公开版本不记录具体内部域名。
- OAuth2 授权入口属于供应商公开能力；本项目 P0 不实现用户授权流程。

### 2.2 Client Credentials（P0）

公开 endpoint 为 `GET /v1/auth/token`，查询参数为 `grant_type=client_credentials`、`client_id` 和 `client_secret`。成功响应 schema 包含 `access_token`、`token_type` 和 `expires_in`，公开材料不保存任何实例值。

`expires_in` 的相对/绝对时间语义必须在获准集成环境确认，不能由公开示例推断为真实租户事实。真实授权、scope 和令牌生命周期均为 `INTEGRATION-PENDING`。

### 2.3 Authorization Code（P2，未实现）

公开文档存在授权码和刷新令牌相关入口；本阶段只保留抽象扩展点，不记录授权实例、用户信息或令牌内容。

## 3. 通用约定

- URI 使用 `/v1[/{area}]/{resource}[/{action}]` 的层级形式。
- GET 参数使用 query；POST、PUT、PATCH 使用 JSON body。
- 分页字段为 `page_size`、`page_index`、`total`、`values`；公开契约默认每页 30 条，最大 100 条，页码从 0 开始。
- 通用错误结构为 `{ "code": "...", "message": "..." }`；真实状态码和权限结果需在获准环境确认。
- 资源 schema 可包含 `id`、引用对象以及 `created_at`、`updated_at` 等字段；本文件不提供实际标识或时间。
- 限流及重试行为以供应商公开文档为准；实际租户阈值、响应头和剩余额度均为 `INTEGRATION-PENDING`。

## 4. P0 工具 → API 映射

| 工具 | HTTP 方法与路径 | 主要参数 | Scope |
|---|---|---|---|
| `pingcode_list_projects` | `GET /v1/pjm/projects` | 项目过滤条件与分页参数 | `pcp:read:pjm:project` |
| `pingcode_get_project` | `GET /v1/pjm/projects/{project_id}` | `project_id`、归档/删除过滤 | `pcp:read:pjm:project` |
| `pingcode_list_work_items` | `GET /v1/pjm/workitems` | 项目、类型、状态、优先级、负责人、关键字与分页 | `pcp:read:pjm:workitem` |
| `pingcode_get_work_item` | `GET /v1/pjm/workitems/{workitem_id}` | `workitem_id`、归档/删除过滤 | `pcp:read:pjm:workitem` |
| `pingcode_create_work_item` | `POST /v1/pjm/workitems` | `project_id`、`type_id`、`title` 及允许的可选字段 | `pcp:write:pjm:workitem` |
| `pingcode_update_work_item` | `PATCH /v1/pjm/workitems/{workitem_id}` | `workitem_id` 与部分更新字段 | `pcp:write:pjm:workitem` |

辅助公开 endpoint：工作项类型、状态、优先级、标签以及项目状态列表均按 `project_id` 查询；完整路径见 `.kiro/reference/pingcode-api-extract-p0.md`。

## 5. 关键接口细则

### 5.1 获取项目

项目列表使用 `GET /v1/pjm/projects`，支持 scope、关键字、成员、类型、时间范围、归档/删除过滤和分页。项目详情使用 `GET /v1/pjm/projects/{project_id}`。响应 schema 可包含项目标识、名称、类型、状态、负责人引用和时间字段；不包含实际项目数据。

### 5.2 获取工作项

工作项列表使用 `GET /v1/pjm/workitems`，支持项目、类型、父项、负责人、状态、优先级、关键字、归档/删除过滤和分页。详情使用 `GET /v1/pjm/workitems/{workitem_id}`。响应 schema 可包含工作项标识、标题、类型、状态、优先级、负责人引用和时间字段；不包含实际工作项数据。

### 5.3 创建工作项

`POST /v1/pjm/workitems` 的必填 body 字段为 `project_id`、`type_id`、`title`；可选字段包括描述、时间、优先级、状态、负责人、父项和属性。类型、状态方案和状态流转必须由授权环境确认，状态 ID 不能凭猜测构造。

### 5.4 更新工作项

`PATCH /v1/pjm/workitems/{workitem_id}` 使用部分更新 body。允许字段以公开 schema 为准；更新不应因网络不确定而自动重复写入。真实写入结果为 `INTEGRATION-PENDING`。

## 6. 验证状态与阻塞项

- `DOC-VERIFIED`：仅表示公开文档存在对应 endpoint、参数、scope 或 schema。
- `INTEGRATION-PENDING`：真实凭据、租户授权、项目范围、资源字段形态、状态流转、限流阈值和读写结果均未在公开材料中确认。
- 自动化 mock、HTTP mock 和 MCP 协议测试不能替代真实集成。
- 真实集成必须在获准隔离环境单独记录为 `INTEGRATION`；公开仓库不保存标识符、操作时间、运行结果或敏感请求信息。

## 7. P1 预览

公开文档还包含 Wiki 空间、页面、正文和版本相关 endpoint。P1 实现前需重新核对参数、scope、正文格式和并发更新语义；本版本不保存任何 Wiki 实例数据。
