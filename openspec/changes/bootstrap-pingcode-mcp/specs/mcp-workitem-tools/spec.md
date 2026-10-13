# Spec Delta

## Purpose

让本地 MCP 客户端通过一致且可发现的工具访问经授权的 PingCode 项目和工作项。工具输入、分页输出与错误结构保持明确，使客户端能够在不接触 REST 细节的情况下安全地读取或执行获准的工作项变更。

## ADDED Requirements

### Requirement: 通过 stdio 提供 MCP 服务

服务 SHALL 使用 MCP stdio 传输；stdout SHALL 仅承载 MCP 协议消息，诊断日志 SHALL 写入 stderr；P0 SHALL NOT 启动共享 HTTP 服务。

#### Scenario: MCP 客户端建立连接并发现工具
- **WHEN** MCP 客户端通过 stdin/stdout 启动服务并完成 initialize 与 list_tools
- **THEN** 服务按 MCP 协议响应，stdout 中不混入启动横幅、日志或其他非协议文本

#### Scenario: 服务输出诊断日志
- **WHEN** 服务记录启动、运行或错误诊断
- **THEN** 日志写入 stderr 并遵循脱敏要求

### Requirement: 注册六个 P0 项目与工作项工具

服务 SHALL 注册 `pingcode_list_projects`、`pingcode_get_project`、`pingcode_list_work_items`、`pingcode_get_work_item`、`pingcode_create_work_item` 和 `pingcode_update_work_item` 六个工具，输入 JSON Schema SHALL 明确参数类型、必填项、限制和允许字段。

#### Scenario: 浏览 P0 工具定义
- **WHEN** 客户端调用 list_tools
- **THEN** 返回以上六个名称；列表工具定义分页字段 `page_size` 与 `page_index`，且工作项筛选字段与 `docs/api-contract.md` §4.1 中已核验参数一致

#### Scenario: 调用项目读取工具
- **WHEN** 客户端调用 `pingcode_list_projects` 或 `pingcode_get_project`
- **THEN** 列表工具接受契约 §5.1 的可选过滤字段与分页字段，详情工具要求非空 `project_id`

#### Scenario: 调用工作项读取工具
- **WHEN** 客户端调用 `pingcode_list_work_items` 或 `pingcode_get_work_item`
- **THEN** 列表工具要求 `project_id` 并可接受契约 §5.3 的过滤字段及分页字段，详情工具要求 `workitem_id` 和用于权限校验的 `project_id`

#### Scenario: 调用工作项创建工具
- **WHEN** 客户端调用 `pingcode_create_work_item`
- **THEN** Schema 要求 `project_id`、`type_id`、`title`，且可选字段仅限 `docs/api-contract.md` §5.5 记录的字段

#### Scenario: 调用工作项更新工具
- **WHEN** 客户端调用 `pingcode_update_work_item`
- **THEN** Schema 要求 `workitem_id` 和用于权限校验的 `project_id`，并仅允许契约 §5.6 中记录的部分更新字段

### Requirement: 返回可识别的分页与资源结果

列表工具 SHALL 返回 `page_size`、`page_index`、`total` 和 `values`；项目及工作项资源结果 SHALL 包含资源 `id`，创建和更新结果 SHALL 包含 PingCode 返回的资源标识。

#### Scenario: 返回项目或工作项列表
- **WHEN** 任一列表工具成功执行
- **THEN** MCP 结果包含当前页 `values`、资源 ID 和完整分页信息

#### Scenario: 返回单项、创建或更新结果
- **WHEN** 任一详情、创建或更新工具成功执行
- **THEN** MCP 结果包含返回资源的 `id` 和契约已记录的关键字段

### Requirement: 返回带稳定错误类型的结构化错误

工具调用失败 SHALL 返回包含稳定 `error.type` 与安全消息的结构化错误；错误类型 SHALL 能区分输入校验、鉴权、权限、未找到、限流、上游服务、超时、只读拒绝和项目未授权。

#### Scenario: 工具调用遇到 API 或策略错误
- **WHEN** 工具调用触发输入、PingCode HTTP 或本地安全策略错误
- **THEN** 客户端收到结构化错误及对应错误类型，而不是只收到不可分类的自由文本

#### Scenario: 工具调用超时或被取消
- **WHEN** 调用的 Context 超时或取消
- **THEN** 客户端收到可识别的超时或取消错误，且工具不会继续执行后续请求

### Requirement: 不暴露删除类工具

P0 MCP 工具集合 SHALL NOT 包含删除项目、工作项或其他资源的工具。

#### Scenario: 查询可用工具
- **WHEN** MCP 客户端调用 list_tools
- **THEN** 返回的工具集合中不存在删除类工具
