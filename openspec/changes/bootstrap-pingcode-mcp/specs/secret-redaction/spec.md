# Spec Delta

## Purpose

保护 PingCode 凭据、访问令牌和工作项敏感内容，避免它们从调试日志、MCP 错误或上游错误转存中泄露。所有诊断信息只保留排查所需的安全类别和状态元数据。

## ADDED Requirements

### Requirement: 凭据与授权头永不写入日志或错误

服务 SHALL NOT 在日志、MCP 响应或错误文本中输出 Client Secret、access token、refresh token、完整 Authorization Header，或包含 OAuth 凭据的原始 URL/查询参数。

#### Scenario: 记录 OAuth 请求失败
- **WHEN** 令牌请求失败且服务记录诊断信息
- **THEN** 输出不包含 `client_secret`、`client_id` 的敏感值、令牌值或未脱敏的请求 URL 查询参数

#### Scenario: 记录带鉴权头的 API 请求失败
- **WHEN** PingCode 资源请求携带 Authorization Header 并发生失败
- **THEN** 日志和错误响应不包含该 Header 或 Bearer Token

### Requirement: 不记录或返回敏感工作项正文

服务 SHALL NOT 将完整工作项描述、属性正文或其他敏感资源正文写入诊断日志或错误消息；错误 SHALL 使用稳定类型、安全摘要和允许披露的状态元数据。

#### Scenario: 上游返回含正文的错误
- **WHEN** PingCode 错误响应或异常响应体中包含工作项正文或敏感字段
- **THEN** 服务丢弃或脱敏正文，仅返回错误类型及安全的状态/错误码信息

#### Scenario: 工作项工具调用失败
- **WHEN** 工作项读取、创建或更新工具失败
- **THEN** 日志记录不包含工具请求中的完整描述、属性正文或上游资源正文

### Requirement: 安全记录可排障的非敏感元数据

服务 MAY 记录请求类别、工具名、HTTP 状态、错误类型、耗时与重试次数等非敏感元数据，但 SHALL 在任何错误路径维持凭据和正文脱敏。

#### Scenario: 记录可恢复的上游错误
- **WHEN** 请求返回 429、5xx、超时或其他分类错误
- **THEN** 日志可包含对应状态、错误类型和重试信息，但不含凭据、完整 URL 查询或响应正文
