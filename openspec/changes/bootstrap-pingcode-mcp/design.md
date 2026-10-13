# Design

> 状态：已获负责人批准；公开版本不记录审批日期或其他内部操作元数据。

## Context

- 仓库已有 Go 1.24 模块、`cmd/pingcode-mcp` 和 `internal/{config,auth,pingcode,mcp,security}` 包占位；当前没有业务实现。
- M1 已在 `docs/api-contract.md` 固化 P0 API 路径、参数、scope、分页、响应和限流约定。该契约来自官方快照；`expires_in` 语义、租户 scope 和写入结果均为 `INTEGRATION-PENDING`。
- `go.mod` 已固定 `github.com/modelcontextprotocol/go-sdk v1.8.0`，目前为 indirect；`.env.example` 已列出全部 P0 环境变量。CI 骨架已执行 gofmt 检查、`go vet ./...`、`go build ./...` 和 `go test ./...`，但目前没有业务测试覆盖。
- 目前没有可用于本设计工作的真实 PingCode 授权；不得把 mock 结果称作联调。变更背景及 Phase 1 范围见 proposal.md。

## Goals / Non-Goals

**Goals:**

- 让每个实现任务可依据五份 spec delta 实现和验收，并遵循 `docs/api-contract.md` §2–§5 的外部 API 行为。
- 让测试可以通过替身 TokenProvider、`httptest` 与 MCP 协议客户端离线运行。
- 在网络请求边界之前完成只读与项目策略检查，并在 REST 客户端增加兜底校验。
- 使 MCP stdio 的工具、错误和分页结果保持稳定，且不将诊断内容写入 stdout。

**Non-Goals:**

- 本设计文档撰写阶段不加入 Go 业务代码；设计获批后由 tasks.md 的实现任务落地。在获得真实凭据与写入授权之前，不实际调用 PingCode、不执行真实写入或 Phase 1 自举验收。
- 不实现 Authorization Code、refresh_token、Streamable HTTP、Wiki、删除、批量写操作或部署服务。
- 不改变 PingCode 官方契约，也不把未核验的字段、scope、状态流转或限流值当作事实。

## Decisions

### 1. 分层与依赖方向

采用 `cmd/` 作为唯一组合根，并沿用现有包边界：

```text
cmd/pingcode-mcp
  ├── internal/config       环境变量读取、默认值、配置校验
  ├── internal/security     项目访问策略、只读策略、脱敏辅助
  ├── internal/auth         TokenProvider 接口与 Client Credentials 实现
  ├── internal/pingcode     REST 请求、资源模型、分页、API 错误
  └── internal/mcp          stdio Server、六个工具及 MCP 错误映射
```

`config` 和 `security` 不依赖业务包；`auth` 不依赖 PingCode Client；`pingcode` 依赖 `auth.TokenProvider`、配置值和安全策略；`mcp` 依赖 `pingcode` 的窄接口与安全策略；只有 `cmd` 同时依赖各实现并负责装配。该方向使令牌和 HTTP mock 能独立测试，也避免 MCP 协议反向进入 REST 层。被否方案：让 MCP handler 直接拼 URL 或让 PingCode Client 创建工具，二者都会把传输、策略与外部契约耦合在一起。

### 2. TokenProvider 接口与刷新策略

在 `internal/auth` 定义：

```go
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}
```

PingCode Client 只依赖此接口，不读取 Client ID/Secret，也不处理 grant 类型。`ClientCredentialsProvider` 从配置拿凭据，严格按 `docs/api-contract.md` §2.2 以 `GET /v1/auth/token` 发送 `grant_type`、`client_id`、`client_secret` 查询参数；不得改成 JSON body。Authorization Code 和 refresh_token 只作为未来可替换 Provider 的扩展点，P0 不发请求（契约 §2.3）。

Provider 缓存 `{access_token, expires_at}`，使用互斥状态和可取消的 refresh-flight 通知让并发调用共享一次刷新；等待者可独立因自己的 Context 取消。调用方取消会终止令牌 HTTP 请求。提前 60 秒刷新，避免在令牌临界到期时发出业务请求。根据契约 §2.2 的风险说明，将 `expires_in` 解析为绝对 Unix 秒级时间；为兼容快照字段标注 `String`、示例却为数字的形态，可接受 JSON 整数或十进制数字字符串，但不按相对秒数换算。该语义仍须真实集成验证，不能以 mock 结果替代。

被否方案：直接缓存字符串且不保留过期时刻，无法安全刷新；把 `expires_in` 当相对秒数与当前官方快照中的示例不符。实现不引入独立 singleflight 依赖，避免依赖膨胀。

### 3. HTTP Client、分页、错误和重试

PingCode Client 用共享 `http.Client`，超时取 `PINGCODE_HTTP_TIMEOUT`（默认 `30s`），每个请求带调用方 Context。路径、请求字段和方法仅按 `docs/api-contract.md` §4.1、§5.1–§5.6 实现；POST/PATCH 用 JSON body。项目与工作项 API 必须保持其 scope 和已记录字段，不从名称推测额外 API。

分页按契约 §3：默认 `page_size=30`、`page_index=0`，最大页大小 `100`；超出上限在本地拒绝。响应保留 `page_size`、`page_index`、`total`、`values`，上层不得丢掉分页总量。

错误映射如下。它表示客户端内部分类，不声称 PingCode 所有租户都必定返回这些状态；契约 §6.1 的无凭据探针实际返回 HTTP 400，仍按一般 API 错误处理。

| 上游响应 | 客户端错误类型 | 处理 |
|---|---|---|
| 401 | `AUTHENTICATION` | 不自动重放当前请求 |
| 403 | `PERMISSION` | 不重试 |
| 404 | `NOT_FOUND` | 不重试 |
| 429 | `RATE_LIMITED` | 读取契约 §3 的 `X-RateLimit-Retry-After` 或 `X-PC-Retry-After`；最多三次总尝试，按头等待；缺失或无效时立即返回 |
| 5xx | `UPSTREAM` | 只对 GET 使用 250ms、500ms 有界指数等待，最多三次总尝试 |
| 其他失败状态 | `UPSTREAM` 或 `VALIDATION` | 安全保留 HTTP 状态和可用 API `code`，不返回原始正文 |

GET 重试等待必须响应 Context 取消；POST 创建与 PATCH 更新没有自动重试，避免响应丢失时重复写入。429 的 header 名称与公有云/私有部署差异以契约 §3 为准；真实 header 值格式和租户限流仍待 `INTEGRATION` 验证。被否方案：所有方法统一重试或忽略 429 头，会造成重复写入或立即触发限流。

REST 错误使用稳定类型和有限元数据；`internal/mcp` 将类型映射到结构化 `error.type`。不把 PingCode 原始错误正文整体放入 MCP 响应。

### 4. 写保护和项目白名单

`internal/security.Policy` 提供项目读取与写入检查；MCP tool handler 在调用 Client 前检查一次，PingCode Client 在生成 HTTP 请求前再检查一次。写入必须同时满足 `PINGCODE_READ_ONLY=false` 和项目 ID 精确位于 `PINGCODE_ALLOWED_PROJECT_IDS`；默认只读、缺少/空白名单均 fail-closed。策略错误分别映射为 `READ_ONLY`、`PROJECT_NOT_ALLOWED`。

项目详情、工作项列表和工作项详情都要求白名单范围；`pingcode_list_work_items` 的 `project_id` 为必填。详情和更新工具要求调用方提供 `project_id` 作为本地授权上下文，读取的资源须校验返回 `project.id` 与该上下文相符后才可输出或更新。工作项更新先校验项目和资源关联，再按契约 §5.6 PATCH。项目列表端点在 API 上没有项目 ID 参数（契约 §5.1），因此本地只返回白名单中的结果，不把其他项目数据放入 MCP 响应；空白名单不返回项目资源。该目录型读取仅服务项目发现，不授权其他项目定向读取。

被否方案：只在 MCP 层做检查可被其他 Client 调用路径绕过；只在 Client 层检查则错误发生太晚且缺少清晰的工具级反馈。任何删除工具或批量写入均不注册。

### 5. MCP Server、工具注册与 Schema

使用仓库现有且已固定的官方 `github.com/modelcontextprotocol/go-sdk`，实际编码前确认 v1.8.0 的公开 API 与示例匹配，并在 `go.mod` 中将它作为直接依赖。按官方 Go SDK 的 stdio server 模式创建 server、注册工具并以 stdio transport 运行；官方示例使用 `mcp.NewServer`、`mcp.AddTool` 与 `mcp.StdioTransport`（[官方 quick start](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/quick_start.md)）。如固定版本 API 有差异，在实现任务内基于该版本的官方文档调整，不改用非官方 SDK。

工具名固定为 `pingcode_list_projects`、`pingcode_get_project`、`pingcode_list_work_items`、`pingcode_get_work_item`、`pingcode_create_work_item`、`pingcode_update_work_item`。Typed input 使用 JSON 字段名、说明和 schema 限制；Schema 中必填项与 `docs/api-contract.md` §4.1、§5.1–§5.6 对齐，同时显式标记本地授权上下文。工作项创建必填 `project_id`、`type_id`、`title`；更新只暴露契约 §5.6 的部分字段。状态 ID 合法性仍受契约 §5.5–§5.6 中 PingCode 状态方案及状态流转约束，不自行补造状态表。

列表工具结果包括资源 ID、`page_size`、`page_index`、`total`、`values`；详情、创建、更新结果包含 PingCode 资源 ID。工具错误包含稳定类型和安全消息。日志器显式写 stderr，stdout 仅交给 SDK stdio transport。被否方案：手写 JSON-RPC transport 会复制 SDK 协议实现；在 stdout 打印日志会破坏 stdio 客户端通信。

### 6. 测试策略

- 单元测试覆盖环境变量默认值/校验、TokenProvider 并发缓存、绝对时间到期刷新、取消和脱敏、分页边界、安全策略和错误映射。
- `httptest` 模拟契约 §2.2、§4.1、§5.1–§5.6 所需的令牌与资源响应；覆盖 401/403/404/429/5xx、API 错误信封、重试头、请求方法/参数及不重试写入。
- MCP 集成测试使用 SDK 客户端与内存传输（若固定版本支持）或子进程 stdio harness，执行 initialize、list_tools、六工具的成功/错误调用，并检查 Schema、分页、结构化错误及 stdout 干净性。
- 所有自动化测试使用替身和 mock，不读取 `.pingcode.env`、环境中的真实 Token 或 Secret，也不连接真实 PingCode。真实租户联调需另立 `INTEGRATION` 记录，注明授权与隔离测试项目；无授权时 FR-012 保持阻塞。

被否方案：只有 handler 单元测试无法验证 MCP 握手与 JSON-RPC 边界；只有真实 PingCode 验证不可重复且不符合无凭据测试要求。

### 7. 配置加载

只读取 `.env.example` 当前列出的环境变量，不要求用户在对话中提供凭据：

| 变量 | 默认/校验 | 用途 |
|---|---|---|
| `PINGCODE_CLIENT_ID` | 未设置时允许离线测试装配；真实 API 首次鉴权前必须配置 | OAuth Client ID |
| `PINGCODE_CLIENT_SECRET` | 未设置时允许离线测试装配；真实 API 首次鉴权前必须配置 | OAuth Client Secret |
| `PINGCODE_BASE_URL` | `https://open.pingcode.com` | PingCode REST 根地址；规范化尾部斜线，私有部署按契约配置 |
| `PINGCODE_READ_ONLY` | 缺省、空值为 `true`；非法布尔值使配置加载失败 | 写入总开关 |
| `PINGCODE_ALLOWED_PROJECT_IDS` | 空值为空白名单；逗号分隔、去空白、拒绝空 ID 项 | 项目授权范围；空白不允许项目定向读写 |
| `PINGCODE_HTTP_TIMEOUT` | `30s`；必须为正 duration | HTTP 超时 |
| `PINGCODE_LOG_LEVEL` | `info`；仅允许支持的标准等级 | stderr 日志等级 |

Client ID/Secret 仅由 `auth` 使用并保存在内存配置中；不得加入错误或日志。配置解析独立于实际 PingCode 调用，因此无凭据时测试可启动。被否方案：测试从 `.pingcode.env` 读取本机状态会让自动化依赖真实秘密并增加泄露面。

## Risks / Trade-offs

- **`expires_in` 实际为相对秒数或响应类型与快照不同** → 按 M1 快照的绝对 Unix 秒解释并覆盖整数/数字字符串；错误时 fail closed；真实令牌响应确认前标记 `INTEGRATION-PENDING`，不得宣称契约已被租户验证。
- **企业令牌具有系统管理员权限**（契约 §2.2）→ 默认只读，所有写入要求显式关闭只读并命中项目白名单；测试与自举限定在隔离项目。
- **项目列表 API 是全局分页接口**（契约 §5.1），本地只能对外返回白名单结果 → 可能仍需拉取多页再过滤，需避免日志和 MCP 响应包含未授权项目；设计不把过滤视为 PingCode 服务端权限替代。
- **mock 与实际 PingCode 状态、权限、限流头或字段有差异** → 固定官方快照为契约依据，保留 `INTEGRATION-PENDING`；获得授权后对隔离租户执行有记录的只读/写入验证。
- **白名单上下文与工作项真实所属项目不一致** → 调用方必须提供 `project_id`，服务校验资源返回的项目关联后才输出或写入；不匹配按权限拒绝处理。
- **首个自举需要真实凭据和隔离项目写权限** → FR-012 作为外部阻塞项单独登记，未完成真实回读前不关闭该任务。

## Migration Plan

本变更仅新增本地 stdio 工具，不需要服务端部署或数据迁移。负责人批准 design gate 后，实现与 mock 验证完成，再由负责人配置环境变量并决定是否进行隔离项目的 `INTEGRATION`。回退时停止本地 MCP 进程并继续使用 PingCode Web/独立 REST 客户端；不需要清理远端数据。不得因实现完成而自动执行真实写入。
