# Spec Delta

## Purpose

以已核验的 API 契约封装 PingCode 项目与工作项 REST 访问，统一请求取消、分页、错误分类和有限重试，使上层工具不需要猜测外部路径或自行处理限流细节。

## ADDED Requirements

### Requirement: 遵守已核验的项目与工作项 API 契约

PingCode 请求 SHALL 使用 `docs/api-contract.md` §2–§5 中记录的根路径、HTTP 方法、资源路径、参数位置、必填字段和 scope；未记载的字段或外部行为 SHALL NOT 被推断后发送。

#### Scenario: 调用 P0 项目与工作项接口
- **WHEN** 服务读取或写入 P0 项目、工作项资源
- **THEN** 项目列表/详情使用 `GET /v1/pjm/projects` 与 `/v1/pjm/projects/{project_id}`，工作项列表/详情/创建/部分更新使用契约 §4.1 所列路径和方法

#### Scenario: 创建或部分更新工作项
- **WHEN** 服务发送创建或更新请求
- **THEN** 创建使用契约 §5.5 中必填的 `project_id`、`type_id`、`title`，部分更新只发送 §5.6 允许的字段，并按契约使用 JSON 请求体

### Requirement: 校验并保留分页信息

列表请求 SHALL 使用 `page_size` 和从 0 开始的 `page_index`；默认页大小 SHALL 为 30，最大 SHALL 为 100；列表响应 SHALL 保留 `page_size`、`page_index`、`total` 与 `values`。

#### Scenario: 使用默认分页参数
- **WHEN** 调用方未提供分页参数
- **THEN** 客户端以 `page_size=30`、`page_index=0` 请求列表，并在响应中保留分页元信息

#### Scenario: 限制页大小
- **WHEN** 调用方提供大于 100 的 `page_size`
- **THEN** 请求在发往 PingCode 前被拒绝为输入校验错误

#### Scenario: 请求后续页面
- **WHEN** 调用方提供有效的 `page_index` 和不大于 100 的 `page_size`
- **THEN** 客户端按原值请求对应页面并返回该页数据及总量

### Requirement: 分类 PingCode HTTP 与 API 错误

客户端 SHALL 将 HTTP 401、403、404、429 和 5xx 分别映射为鉴权、权限、未找到、限流和上游服务错误；其他失败状态 SHALL 以安全的上游/API 错误返回，并可保留脱敏后的 API `code`。

#### Scenario: 返回明确的 HTTP 错误类别
- **WHEN** PingCode 返回 401、403、404、429 或 5xx
- **THEN** 客户端返回与状态对应的稳定错误类型，不把错误响应正文原样暴露给调用者

#### Scenario: PingCode 返回其他错误状态
- **WHEN** PingCode 返回除上述状态外的失败状态或 `{code,message}` 错误信封
- **THEN** 客户端返回上游/API 错误并安全保留可用的状态码和错误码；不把未凭据探针的 HTTP 400 误报为鉴权成功或已验证行为

### Requirement: 仅对幂等 GET 执行有界重试

客户端 SHALL 仅对幂等 GET 在可重试的限流或暂时性服务失败时进行有限次重试；429 SHALL 优先遵守 `docs/api-contract.md` §3 中的 `X-RateLimit-Retry-After` 或 `X-PC-Retry-After`，5xx SHALL 使用有上限的指数退避；POST 与 PATCH SHALL NOT 被盲目自动重试。

#### Scenario: GET 遇到公有云或私有部署限流
- **WHEN** GET 收到 429 且响应含契约定义的重试头
- **THEN** 客户端读取对应重试间隔，在有限重试预算内等待后重试，不立即再次请求

#### Scenario: GET 遇到暂时性 5xx
- **WHEN** GET 收到 5xx
- **THEN** 客户端按有界指数退避进行有限次重试，耗尽预算后返回上游服务错误

#### Scenario: 写请求遇到可重试状态
- **WHEN** POST 创建或 PATCH 更新收到 429 或 5xx
- **THEN** 客户端不自动重放请求，而是返回分类错误供调用方处理

### Requirement: 传播请求取消与超时

所有 PingCode HTTP 请求和退避等待 SHALL 遵循调用方 Context 与已配置的 HTTP 超时；取消或超时 SHALL 终止当前操作并返回可识别错误。

#### Scenario: Context 在请求期间取消
- **WHEN** 调用方取消 Context 或达到请求超时
- **THEN** 当前请求或重试等待尽快终止，且不会启动后续重试
