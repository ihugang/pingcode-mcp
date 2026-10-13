# Proposal

## Why

项目已完成 P0 官方 API 契约整理和 Go 分层骨架，但尚不能通过 MCP 访问 PingCode。现在需要交付稳定、可测试且默认只读的 stdio 服务，使开发者能在人工评审 OpenSpec 后使用受控工具管理隔离测试项目，并为后续自举建立可靠基线（PRD §1、§6；FR-001..FR-012）。

## What Changes

- 实现 Client Credentials 企业令牌提供器、PingCode REST 客户端及限流、分页、错误分类和脱敏处理。
- 通过官方 Go MCP SDK 暴露六个 P0 工具：列出/读取项目，列出/读取工作项，以及创建/更新工作项。
- 增加默认只读开关、项目白名单、无删除工具的写入护栏，以及单元、HTTP mock、MCP 协议测试和 CI 质量检查。
- 提供本地环境变量配置、构建和 Codex stdio 接入说明；真实 PingCode 联调与首次自举须在获得授权后另行完成。

## Capabilities

### New Capabilities

- `auth-token-provider`: 以 Client Credentials 获取、缓存和刷新企业令牌，并安全处理取消和鉴权错误；为未来授权方式保留接口边界。
- `pingcode-rest-client`: 按已核验的 PingCode API 契约执行 REST 请求、分页、错误分类和受限重试。
- `mcp-workitem-tools`: 通过 stdio 暴露六个 P0 项目与工作项工具及稳定的输入、输出和错误约定。
- `write-safety`: 在只读配置和项目白名单约束下控制项目数据访问与工作项写入，不暴露删除能力。
- `secret-redaction`: 阻止 Token、Secret、Authorization Header 及敏感工作项正文进入日志或错误输出。

### Modified Capabilities

无。当前 `openspec/specs/` 尚无已定义 capability；本变更新增以上五项。

## 自举里程碑

- Phase 0：通过 PingCode Web 或官方 REST API 人工建立项目和需求，初始化 Git/OpenSpec 与项目映射。
- Phase 1：本变更覆盖；实现 P0，经负责人评审后，在隔离测试项目完成 MCP 创建、查询、更新和回读验收。
- Phase 2：使用稳定版 MCP 管理下一阶段开发，并将开发版限制到测试项目。
- Phase 3：形成需求、OpenSpec、代码测试、评审、PingCode 回写和归档的完整闭环。

## Impact

- 影响现有 Go 骨架下的 `cmd/` 与 `internal/{config,auth,pingcode,mcp,security}`，并新增相应源码和测试、CI 与接入文档。
- 将现有 `go.mod` 中已固定为 `v1.8.0` 的官方 `github.com/modelcontextprotocol/go-sdk` 从间接依赖转为实际使用，并由实现任务核验兼容性；只增加实现所需的最少依赖。
- 运行形态仅为本地 MCP stdio 进程，不包含服务端部署、共享 HTTP、多租户或 Authorization Code 登录。
- 所有 PingCode 外部行为以 `docs/api-contract.md` 为准；当前无凭据，真实授权、写入和第一次自举仍为 `INTEGRATION-PENDING`。
