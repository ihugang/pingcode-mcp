# Spec Delta

## Purpose

为 PingCode P0 请求提供不依赖用户交互的企业访问令牌，并在进程内安全复用与刷新令牌。该能力仅覆盖 Client Credentials，保留未来替换令牌提供方式的边界，但不实现用户授权流程。

## ADDED Requirements

### Requirement: 使用 Client Credentials 获取企业令牌

P0 服务 SHALL 按 `docs/api-contract.md` §2.2 使用 `GET /v1/auth/token` 和 `grant_type=client_credentials` 获取企业令牌；Authorization Code 与 refresh_token 流程 SHALL NOT 在本变更中实现或发起。

#### Scenario: 首次获取企业令牌
- **WHEN** 服务首次需要 PingCode 访问令牌
- **THEN** 它按 API 契约将 `grant_type`、`client_id`、`client_secret` 作为查询参数请求令牌，并仅从环境变量读取凭据

#### Scenario: Authorization Code 尚未实现
- **WHEN** P0 服务启动或请求令牌
- **THEN** 它不打开浏览器、不要求用户登录，也不请求 authorization_code 或 refresh_token grant

### Requirement: 并发安全地缓存并提前刷新令牌

令牌提供器 SHALL 在进程内并发安全地复用尚未进入到期安全窗口的令牌，并 SHALL 在到期前刷新；按 `docs/api-contract.md` §2.2 将 `expires_in` 作为绝对 Unix 秒级到期时刻解释，此语义仍为 `INTEGRATION-PENDING`。

#### Scenario: 并发请求共用有效令牌
- **WHEN** 多个调用同时请求有效期内的令牌
- **THEN** 调用方获得同一缓存令牌，且不会因并发刷新产生竞态或重复暴露令牌

#### Scenario: 令牌进入到期窗口
- **WHEN** 缓存令牌达到实现定义的提前刷新窗口或已过期
- **THEN** 提供器先获取新令牌，再将其用于后续调用；若刷新失败则返回脱敏鉴权错误，不把已过期令牌当作有效令牌

#### Scenario: 到期值无法按绝对时间解析
- **WHEN** PingCode 返回的 `expires_in` 缺失、无效或不是可接受的绝对 Unix 秒级时刻
- **THEN** 提供器返回脱敏鉴权错误，且不静默按相对秒数换算

### Requirement: 传播取消并安全返回鉴权错误

令牌请求 SHALL 传播调用方 Context；鉴权失败 SHALL 提供可分类的错误信息，同时 SHALL NOT 在日志或错误文本中包含凭据、令牌或 Authorization Header。

#### Scenario: 调用方取消令牌请求
- **WHEN** 获取令牌期间调用方 Context 被取消或超时
- **THEN** 上游请求尽快结束，并将取消或超时以可识别错误返回

#### Scenario: PingCode 拒绝令牌请求
- **WHEN** 令牌端点返回失败状态或错误信封
- **THEN** 调用方得到鉴权类别错误和安全的状态信息，且错误不包含 Client Secret、Token 或原始敏感请求内容
