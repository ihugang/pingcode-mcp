# PingCode MCP Server (Go) — PRD v1.0

> 状态：待评审 | 日期：未公开 | 项目代号：pingcode-mcp-go

## 1. 背景与目标

开发一个基于 Go 的 PingCode MCP Server，让 Codex CLI 等 MCP 客户端通过 PingCode 官方 REST API 管理项目、需求、工作项和 Wiki。**本项目必须使用 PingCode + OpenSpec + Git 管理自身开发，并通过逐步自举验证产品能力。**

### 1.1 产品目标
- 支持标准 MCP `stdio` 传输，首先服务本地 Codex CLI；后续扩展 Streamable HTTP。
- 第一阶段使用 OAuth 2.0 Client Credentials；预留 Authorization Code 的抽象接口，不在 MVP 中实现用户授权流程。
- 可靠访问 PingCode 项目、工作项；后续支持 Wiki、关联关系、版本。
- 安全、可测试、可审计；默认最小权限和只读模式可配置。

### 1.2 工程目标：自举
- PingCode：业务需求、优先级、状态、验收、缺陷、评审记录的权威来源。
- OpenSpec + Git：技术 Spec、设计、实施计划及代码的权威来源。
- MCP 初版上线后，用**稳定版本** MCP 操作本项目的 PingCode 任务，开发下一版本。
- 始终保留 PingCode Web/独立 REST 客户端作为故障恢复通道。

### 1.3 非目标（MVP）
- 不实现完整 PingCode Web UI；不代替 PingCode 的权限系统。
- 不由 MCP 生成 Spec（由 Codex 完成）；MCP 只提供受控数据接口。
- 不实现浏览器模拟登录、凭据抓取、多租户共享 HTTP 服务。
- 不自动批准业务需求、不执行破坏性批量修改。

## 2. 目标用户与场景
- 个人开发者：Codex 读取需求、生成 OpenSpec Change、创建任务并回写状态。
- 10 人以内异地研发团队：产品经理在 PingCode 维护需求，工程师通过 Agent 执行。
- 后台自动化 Agent：使用独立应用凭据执行获准的任务同步。

核心用例：
1. Codex 列出本项目及待办工作项。
2. Codex 根据某个需求创建 OpenSpec 变更，人工评审后开发。
3. Codex 通过 MCP 创建子任务、更新状态，并写入 Spec/提交链接。
4. 回归测试验证 MCP 仍能管理自己的测试工作项。

## 3. 功能需求

### P0 — MVP / 第一次自举
| ID | 需求 | 验收标准 |
|---|---|---|
| FR-001 | 官方 API 研究与契约验证 | 记录官方鉴权端点、参数、scope、API 路径、分页、响应示例；不凭猜测实现 |
| FR-002 | Client Credentials 鉴权 | 可获取并缓存 Token；到期前刷新；失败时明确报错且不泄露 Secret |
| FR-003 | MCP stdio Server 与客户端接入 | Codex 等支持 MCP stdio 的客户端能列出并调用工具；提供 Kiro 等 Agent Skills 兼容 IDE 的 Skill 安装与 MCP 接入说明；stdout 只用于 MCP 协议 |
| FR-004 | `pingcode_list_projects` | 可分页列出有权限的项目 |
| FR-005 | `pingcode_get_project` | 按 ID 获取项目详情；不存在时明确提示 |
| FR-006 | `pingcode_list_work_items` | 按项目/类型/状态过滤并分页；只返回授权数据 |
| FR-007 | `pingcode_get_work_item` | 获取工作项详情和关键字段 |
| FR-008 | `pingcode_create_work_item` | 在指定项目创建工作项，字段与类型以真实 API 为准；写入需受控 |
| FR-009 | `pingcode_update_work_item` | 修改允许的字段，状态流转须遵循 PingCode 约束 |
| FR-010 | 错误与安全 | 401/403/404/429/5xx 分类；超时、重试、脱敏日志；默认禁止删除 |
| FR-011 | 自动化测试 | 单元、HTTP mock、MCP 协议测试；CI 执行 `go test ./...` |
| FR-012 | 第一次自举验收 | 稳定版 MCP 在隔离测试项目创建、查询、更新一个工作项并回读验证 |

### P1 — Spec / Wiki 闭环
| ID | 需求 | 验收标准 |
|---|---|---|
| FR-101 | `pingcode_list_wiki_pages` | 按空间或父页面分页查询 |
| FR-102 | `pingcode_get_wiki_page` | 获取正文与版本标识，记录正文格式 |
| FR-103 | `pingcode_create_wiki_page` | 创建页面，验证正文格式与编码 |
| FR-104 | `pingcode_update_wiki_page` | 更新前检查版本/哈希；冲突时拒绝静默覆盖 |
| FR-105 | Wiki 历史版本 | 查询版本及指定版本详情；恢复操作需额外审批 |
| FR-106 | 需求—Spec 追踪 | PingCode 工作项保存 Git/OpenSpec 路径或 URL，映射可逆查询 |
| FR-107 | 第二次自举 | 用 MCP 为本项目新增 Wiki 功能需求/任务，执行 OpenSpec 流程并回写证据 |

### P2 — 团队化
- Authorization Code：每个用户单独授权，Token 安全存储、刷新、撤销；确认 PingCode 实际授权语义。
- Streamable HTTP：鉴权、会话隔离、部署、审计、限流。
- Webhook / 增量同步：仅在官方 API 支持并经验证后实现。
- 更细粒度的写入策略、项目白名单、审计导出。

## 4. 技术架构

```text
Codex / Claude / Qoder
        │ MCP stdio (MVP)
        ▼
Go MCP Server ── Tool Handlers ── Application Services
                                         │
                             PingCode REST Client
                                         │
                              OAuth Token Provider
                                         │
                                  PingCode API
```

技术约束：
- Go 使用受支持的稳定版本；MCP SDK 使用官方 `github.com/modelcontextprotocol/go-sdk` 的稳定版本并锁定依赖。
- 分层：`cmd/`、`internal/config/`、`internal/auth/`、`internal/pingcode/`、`internal/mcp/`、`internal/security/`。
- `TokenProvider` 接口独立于 PingCode Client；可替换 Client Credentials / Authorization Code。
- API Client 统一实现请求超时、Context 取消、限流、429 Retry-After、有限次指数退避；不可对非幂等写入盲目重试。
- 参数输入 JSON Schema 明确；响应简洁且包含 ID、分页游标/页码、错误类型。
- 写操作支持 `PINGCODE_READ_ONLY=true` 与 `PINGCODE_ALLOWED_PROJECT_IDS`；禁止绕过项目限制。
- 敏感信息只从环境变量或安全 Secret Store 读取；不提交 `.env`。

## 5. API 契约验证门槛

PingCode 开放平台主页可访问，但此 PRD **不把未验证的具体 URL、字段、OAuth token endpoint、scope、Wiki 正文格式当作事实**。Codex 必须先完成 `docs/api-contract.md`，记录官方文档证据与最小真实请求结果。若官方文档无法取得，暂停依赖该接口的实现，列出阻塞项；不得杜撰。

至少验证：
- Client Credentials 是否对所需项目/工作项/Wiki 接口授权。
- Token URL、请求 Content-Type、Client ID/Secret 提交方式、expires_in、scope。
- 工作项列表/详情/创建/更新的真实路径、字段、分页、权限及状态流转。
- Wiki 页面正文格式、版本/并发更新语义（P1）。

## 6. 自举与工作流

**Phase 0 — 无 MCP**：在 PingCode Web 或官方 REST API 中人工建立本项目及首批需求；初始化 Git/OpenSpec；建立项目映射文件 `docs/pingcode-project.yaml`（仅真实 ID，不含凭据）。

**Phase 1 — 最小闭环**：创建 OpenSpec Change `bootstrap-pingcode-mcp`，经人工 Review 后实现 P0；测试环境运行 MCP；成功创建/读取/更新本项目隔离测试工作项。

**Phase 2 — 稳定版管理开发版**：将稳定版二进制注册为 `pingcode`；开发版注册为 `pingcode-dev`，默认只连测试项目；用稳定版管理 Wiki P1 任务。

**Phase 3 — 完整闭环**：需求 → OpenSpec Proposal/Design/Tasks → 人工批准 → 代码与测试 → Review → PingCode 回写 → 验收 → OpenSpec Archive。

禁止在没有真实授权/真实项目 ID 的情况下声称完成 PingCode 项目创建或自举验收。

## 7. 验收与质量门槛
- `go test ./...`、`go vet ./...` 通过；鉴权、分页、异常、权限拒绝、Token 刷新均有测试。
- MCP 客户端能成功完成 initialize/list_tools/call_tool。
- 不记录 Token、Secret、Authorization Header 或敏感工作项正文到日志。
- 无凭据时仍可执行 mock 测试，真实联调明确标注 `INTEGRATION`。
- 真实自举演示记录：工作项 ID、操作时间、成功结果、回读验证（不含敏感数据）。
- 所有实现必须可追溯到 PingCode 需求 ID 与 OpenSpec Change。

## 8. 交付物
1. 可运行 Go 源码、单元测试、CI。
2. `README.md`：配置、构建、Codex MCP 接入、常见错误。
3. `docs/api-contract.md`：验证过的 PingCode API 契约。
4. `docs/BOOTSTRAP.md`：项目创建、自举演示、回退步骤。
5. `AGENTS.md`、`openspec/` 规范与首个 Change。
6. `.env.example`（仅变量名与占位符）。

## 9. 参考资料
- PingCode 开放平台：https://open.pingcode.com/
- 官方 MCP Go SDK：https://github.com/modelcontextprotocol/go-sdk
- OpenSpec：https://github.com/Fission-AI/OpenSpec
- 社区参考实现（仅借鉴）：https://github.com/ratatatat1/pingcode-mcp
