# PingCode 公开 REST API 契约摘要（P0）

> 发布状态：`public-contract-only`。本文件仅保留公开 endpoint、参数、scope 和响应 schema 的泛化说明；不包含任何内部租户、项目、工作项、运行快照、操作时间、内部链接、本机路径或凭据。
>
> 配套数据文件：`pingcode-api_data.js`。该文件同样是脱敏后的公开契约摘要，不是任何租户的运行导出，也不记录下载或操作时间。

## 1. 公共根路径与通用约定

- 公有云 REST 根路径：`https://open.pingcode.com`。
- REST 资源使用 `/v1/...` 路径；私有部署地址由部署方配置，不在公开材料中记录具体内部域名。
- 请求使用 JSON 资源模型。查询参数用于 GET；JSON body 用于 POST、PUT、PATCH。
- 分页字段：`page_size`、`page_index`、`total`、`values`。公开契约默认每页 30 条，最大 100 条，页码从 0 开始。
- 通用错误结构为 `{ "code": "...", "message": "..." }`；具体状态和字段仍应以供应商公开文档及实际授权环境为准。
- 公开 schema 中的 `id`、`*_id`、`created_at`、`updated_at` 等仅表示字段定义，不提供任何真实值或时间。

## 2. 鉴权契约

### `GET /v1/auth/token?grant_type=client_credentials`

**证据组**：客户端凭据。请求查询参数：

| 字段 | 类型 | 必填 |
|---|---|---|
| `grant_type` | String | 是，固定为 `client_credentials` |
| `client_id` | String | 是 |
| `client_secret` | String | 是 |

成功响应只记录字段 schema：`access_token`、`token_type`、`expires_in`。公开材料不提供访问令牌、密钥、请求头内容或响应实例。`expires_in` 的实际时间语义应在获准集成环境确认，不能以本文件代替真实联调。

## 3. P0 工具与 endpoint 映射

| 工具 | 方法与路径 | Scope | 公开响应摘要 |
|---|---|---|---|
| `pingcode_list_projects` | `GET /v1/pjm/projects` | `pcp:read:pjm:project` | 分页对象，`values` 为项目资源 |
| `pingcode_get_project` | `GET /v1/pjm/projects/{project_id}` | `pcp:read:pjm:project` | 项目资源，含 `id`、`name`、`identifier`、`type`、状态和负责人引用 |
| `pingcode_list_work_items` | `GET /v1/pjm/workitems` | `pcp:read:pjm:workitem` | 分页对象，`values` 为工作项资源 |
| `pingcode_get_work_item` | `GET /v1/pjm/workitems/{workitem_id}` | `pcp:read:pjm:workitem` | 工作项资源，含 `id`、`identifier`、`title`、类型、状态和优先级引用 |
| `pingcode_create_work_item` | `POST /v1/pjm/workitems` | `pcp:write:pjm:workitem` | 创建后的工作项资源 |
| `pingcode_update_work_item` | `PATCH /v1/pjm/workitems/{workitem_id}` | `pcp:write:pjm:workitem` | 更新后的工作项资源 |

## 4. 项目接口 schema

### 项目列表

`GET /v1/pjm/projects` 支持 `scope_type`、`scope_id`、`keywords`、`member_type`、`member_id`、`type`、`created_between`、`updated_between`、`include_deleted`、`include_archived` 以及通用分页参数。`member_type` 与 `member_id` 应成对使用。证据：`项目-获取项目列表`。

### 项目详情

`GET /v1/pjm/projects/{project_id}` 使用路径参数 `project_id`，可选 `include_deleted` 和 `include_archived`。公开响应字段包括 `id`、`name`、`identifier`、`type`、`state`、`assignee`、`created_at`、`updated_at`。证据：`项目-获取一个项目`。

## 5. 工作项接口 schema

### 工作项列表

`GET /v1/pjm/workitems` 支持 `identifier`、`project_id`、`type_id`、`parent_id`、`assignee_id`、`state_id`、`priority_id`、`keywords`、`include_deleted`、`include_archived` 以及通用分页参数。公开响应保留分页外壳，不包含任何实际资源实例。证据：`工作项-获取工作项列表`。

### 工作项详情

`GET /v1/pjm/workitems/{workitem_id}` 使用路径参数 `workitem_id`，可选 `include_deleted` 和 `include_archived`。公开响应字段包括 `id`、`identifier`、`title`、`type`、`state`、`priority`、`assignee`、`created_at`、`updated_at`。证据：`工作项-获取一个工作项`。

### 创建工作项

`POST /v1/pjm/workitems` 的必填 body 字段为 `project_id`、`type_id`、`title`；可选字段包括 `description`、`start_at`、`end_at`、`priority_id`、`state_id`、`assignee_id`、`parent_id` 和 `properties`。具体类型、状态方案和状态流转须由 PingCode 授权环境返回并确认。证据：`工作项-创建一个工作项`。

### 部分更新工作项

`PATCH /v1/pjm/workitems/{workitem_id}` 使用路径参数 `workitem_id`，body 为部分更新字段，可更新 `title`、`description`、`start_at`、`end_at`、`priority_id`、`state_id`、`assignee_id`、`parent_id` 和 `properties`。证据：`工作项-部分更新一个工作项`。

## 6. 辅助 endpoint

以下只登记公开路径和用途，不包含任何实例数据：

- `GET /v1/pjm/workitem/workitem_types?project_id={project_id}`：工作项类型。
- `GET /v1/pjm/workitem/workitem_states?project_id={project_id}&workitem_type_id={workitem_type_id}`：工作项状态。
- `GET /v1/pjm/workitem/workitem_priorities?project_id={project_id}`：工作项优先级。
- `GET /v1/pjm/workitem/tags?project_id={project_id}`：工作项标签。
- `GET /v1/pjm/project/states?project_id={project_id}`：项目状态。

## 7. 公开性结论与人工 Gate

- 两份 `.kiro/reference/` 文件现仅作为公开契约摘要，不应被解释为租户数据、运行日志或联调证据。
- 真实租户的授权范围、资源标识、时间、状态流转、限流阈值和写入结果均不纳入公开仓库。
- 任何真实集成必须在隔离环境中单独记录为 `INTEGRATION`，并在发布前由负责人确认是否可以公开摘要。
- 公开仓库不保存凭据、访问令牌、敏感请求头、内部域名、内部链接或本机路径。
