# Codex Bootstrap Prompt — PingCode MCP Server (Go)

> 将下面“任务指令”整体交给在项目根目录运行的 Codex CLI。先把同目录 `PRD.md` 放入项目 `docs/PRD.md`。

## 任务指令

你是本项目的主开发 Agent，同时承担启动阶段的项目工程化工作。项目名称 `pingcode-mcp-go`。请严格依据 `docs/PRD.md` 实施。**核心目标是用 PingCode + OpenSpec + Git 管理开发 PingCode MCP Server，并实现逐步自举。**

### 工作原则
1. **先验证，后实现。** 阅读 PingCode 官方开放平台 `https://open.pingcode.com/` 的当前 API 文档，核对 OAuth 2.0 Client Credentials 的实际端点、字段、权限、分页、项目/工作项 CRUD。无法验证的接口不得猜测；形成 `docs/api-contract.md`，列出证据与待确认项。
2. **Spec-Driven。** 检查本机 OpenSpec CLI 版本和官方使用方式，初始化 OpenSpec；建立首个 `bootstrap-pingcode-mcp` Change，包含 proposal/design/tasks/spec delta。若 CLI 命令与示例不一致，以已安装版本和官方文档为准。
3. **阶段性审批。** 在完成 API 契约核验与首个 Proposal/Design/Tasks 后，先输出评审摘要并等待我的确认，再开展大规模编码。可以提前搭建无副作用的骨架和 mock 测试。
4. **安全优先。** 不要求我把 Client Secret 粘贴进对话；只读取安全注入的环境变量。绝不打印 Token；不把 Secret 写入 Git、日志或测试快照。真实写入 PingCode 前，确认目标项目 ID、权限与写入范围；禁止生产环境破坏性操作。
5. **不要伪造完成度。** 若无 API 凭据或无法访问官方文档，完成可验证的 mock 部分，明确阻塞项，不得声称真实联调成功。

### Step 1 — 仓库检查与初始化
- 检查现有目录，保留已有文件和改动；若为空，初始化 Go module、Git、README、`.gitignore`、`.env.example`。
- 选择当前稳定、兼容的 Go 与官方 MCP Go SDK 版本，锁定依赖；优先 stdio，不实现 HTTP。
- 建立 `AGENTS.md`：Spec 先行、需求追踪、测试、代码 Review、最小权限、禁止 Agent 自批需求。
- 建立 `docs/PRD.md`、`docs/BOOTSTRAP.md`、`docs/api-contract.md` 和项目 ID 映射模板。

### Step 2 — PingCode API 调研（第一道 Gate）
- 检查官方 API 文档：OAuth Client Credentials 的 Token Endpoint、认证参数、scope、失效/续期、错误返回。
- 检查项目列表/详情、工作项列表/详情/创建/更新的实际 API 定义、字段类型、分页、过滤条件、状态变更语义。
- 对每个 P0 Tool 建立 `工具名 → API 路径 → 请求参数 → 响应字段 → 权限 → 官方证据` 表。
- 如有凭据，先执行**只读**联调；写入联调需获得我的明确授权并限定测试项目。
- 输出发现、差异和阻塞项；不要将未经核验的路径硬编码。

### Step 3 — OpenSpec（第二道 Gate）
- 初始化 OpenSpec，并创建 `bootstrap-pingcode-mcp` 变更。
- Proposal 说明为何开发、目标、非目标和自举里程碑。
- Design 明确 TokenProvider、HTTP Client、MCP Handler 分层、错误处理、限流与写入安全策略。
- Tasks 按可验收的小任务拆分，关联 PRD FR-001 至 FR-012。
- Spec delta 用可测试的 SHALL / WHEN / THEN 约束描述能力。
- 提交评审摘要，**等待我批准设计后再进行大规模实现**。

### Step 4 — P0 实现（设计批准后）
- 实现 OAuth Client Credentials TokenProvider：并发安全缓存、到期刷新、Context 取消、脱敏错误。
- 实现 PingCode REST Client：基础 URL 配置、超时、分页、状态码处理、429 重试、只读保护、项目白名单。
- 使用官方 Go MCP SDK 实现 stdio Server 和以下 Tools：
  `pingcode_list_projects`、`pingcode_get_project`、`pingcode_list_work_items`、`pingcode_get_work_item`、`pingcode_create_work_item`、`pingcode_update_work_item`。
- 所有工具提供清晰 JSON Schema、结构化错误、分页支持；默认不提供删除类工具。
- 写单元测试、httptest mock、MCP 协议集成测试；使用测试替身，不依赖真实 PingCode 凭据。
- 运行 `go fmt`、`go vet ./...`、`go test ./...`，修复失败。

### Step 5 — 第一次自举验收（需真实凭据及写入授权）
- 先通过 Web 或独立 REST API 建立 PingCode 项目，取得真实项目 ID；不能自动完成则列出最少人工操作。
- 用稳定版 MCP 在**隔离测试项目**创建一个测试工作项，读取、更新并再次读取验证。
- 记录证据（工作项 ID、状态、操作摘要）；敏感数据脱敏。
- 注册稳定版为 `pingcode`；开发版单独注册为 `pingcode-dev`，并限制到测试项目。
- 之后通过稳定版 MCP 管理下一阶段 Wiki/Spec 功能开发任务，形成真正的自举。

### Step 6 — P1/P2 规划（暂不擅自实现）
- P1：Wiki 页面列表、读取、创建、更新、版本冲突检测、需求与 Git/OpenSpec 关联。
- P2：Authorization Code、Streamable HTTP、多人权限隔离、审计和 Webhook。
- 对 Wiki 格式、版本控制和 OAuth 用户授权语义，必须重新查证官方 API。

### 交付与汇报格式
每完成一个阶段，输出：
1. 已完成的真实文件与改动。
2. 测试命令及结果。
3. 对应 PingCode 需求 ID（未创建则标记 `PENDING`）。
4. 对应 OpenSpec Change/Task。
5. 风险与阻塞项。
6. 下一阶段计划，以及需要我批准的动作。

**现在先执行 Step 1–3，完成文档和设计 Gate；不要跳过评审直接进入全面开发。**
