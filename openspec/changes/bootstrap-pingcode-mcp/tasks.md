# Tasks

> 追踪说明：本清单以 PRD FR 编号和本 change 的 capability/spec 作为实现追踪；尚未创建的 PingCode 需求 ID 一律为 `PENDING`。标记 `[x]` 的项表示实现或离线验证已完成；公开版本不记录内部提交哈希、审批日期或真实租户运行元数据。

## 1. 先决产物（已完成）

- [x] 1.1 完成 P0 官方 API 调研并记录端点、参数、scope、分页、错误和限流约定（关联 FR-001；Spec：`pingcode-rest-client`；验收方式：核对 `docs/api-contract.md` §2–§5 与官方快照证据，未核验行为保留 `INTEGRATION-PENDING`；PingCode ID：`PENDING`；产物：`docs/api-contract.md`，M1 完成）
- [x] 1.2 建立 CI 骨架并接入格式检查、`go vet ./...`、`go build ./...`、`go test ./...`（关联 FR-011；Spec：五项 capability；验收方式：核对 `.github/workflows/ci.yml` 已执行以上命令；业务测试矩阵仍未完成；PingCode ID：`PENDING`；M0 完成）

## 2. 配置、鉴权与 REST Client

- [x] 2.1 实现环境变量加载与 fail-closed 校验，覆盖只读默认值、空白名单、超时和日志等级，并加入配置单元测试（关联 FR-002、FR-010；Spec：`write-safety`、`secret-redaction`；验收方式：无凭据测试配置通过，缺省只读为 true，非法配置拒绝启动，运行配置单元测试；PingCode ID：`PENDING`；已完成：离线实现与配置测试通过）
- [x] 2.2 实现 `TokenProvider` Client Credentials、并发缓存、提前刷新、绝对 Unix `expires_in` 解析、Context 取消和脱敏错误，并加入成功/失败/竞态单元测试（关联 FR-002、FR-010、FR-011；Spec：`auth-token-provider`、`secret-redaction`；验收方式：fake HTTP transport 验证契约 §2.2 查询参数、绝对时间刷新、取消和无凭据泄露；PingCode ID：`PENDING`；已完成：离线令牌实现与测试通过）
- [x] 2.3 实现项目与工作项 REST 方法、契约字段映射、分页边界和 HTTP 错误分类，使用 `httptest` 覆盖成功、400、401、403、404 与 5xx（关联 FR-004、FR-005、FR-006、FR-007、FR-008、FR-009、FR-010、FR-011；Spec：`pingcode-rest-client`；验收方式：断言契约 §4.1、§5.1–§5.6 的方法/路径/参数/字段，默认页大小 30、最大 100、页索引从 0，并运行 HTTP mock 测试；PingCode ID：`PENDING`；已完成：离线 HTTP mock 测试通过）
- [x] 2.4 实现仅 GET 的有限重试、429 重试头处理、Context-aware 等待与脱敏 API 错误，并用 `httptest` 验证写请求不重放（关联 FR-010、FR-011；Spec：`pingcode-rest-client`、`secret-redaction`；验收方式：覆盖契约 §3 两类重试头、5xx 退避、取消等待、POST/PATCH 不重试和错误正文脱敏；PingCode ID：`PENDING`；已完成：离线 HTTP mock 测试通过）

## 3. 安全策略

- [x] 3.1 实现工具层前置与 Client 层兜底的只读/项目白名单策略，校验工作项所属项目，不提供删除或批量写入，并加入权限边界单元测试（关联 FR-006、FR-007、FR-008、FR-009、FR-010、FR-011；Spec：`write-safety`、`secret-redaction`；验收方式：覆盖缺省只读、显式只读、非白名单读写、空白名单、工作项项目不匹配、单项写入及删除能力不存在；PingCode ID：`PENDING`；已完成：Client 层兜底与归属校验；工具层策略与 MCP 协议测试通过）

## 4. MCP 工具与协议测试

- [x] 4.1 使用官方 Go MCP SDK 实现 stdio Server 并注册六个 P0 工具，定义可核验 JSON Schema、结构化错误及含 ID/分页信息的结果（关联 FR-003、FR-004、FR-005、FR-006、FR-007、FR-008、FR-009、FR-010；Spec：`mcp-workitem-tools`、`write-safety`；验收方式：Schema 测试核对工具名、必填项、字段限制和无删除工具，handler 单测验证错误及分页响应；PingCode ID：`PENDING`；已完成：离线 MCP 实现与 Schema/handler 测试通过）
- [x] 4.2 编写 MCP 协议集成测试，使用内存传输或 stdio harness 完成 initialize、list_tools、六工具 call_tool 成功与拒绝路径（关联 FR-003、FR-011；Spec：`mcp-workitem-tools`、`secret-redaction`；验收方式：mock PingCode 端点、不需要凭据，验证 stdout 仅含协议、日志写 stderr、结构化错误和 Context 取消；PingCode ID：`PENDING`；已完成：离线 MCP 协议测试通过）

## 5. 交付与验收

- [x] 5.1 更新 README 与 `.env.example` 注释，说明安全配置、构建、Codex stdio 接入、工具和常见错误，并补充 `docs/BOOTSTRAP.md` 的授权前置与回退步骤（关联 FR-003、FR-010、FR-012；Spec：`mcp-workitem-tools`、`write-safety`；验收方式：按文档命令在无凭据环境完成配置说明检查，示例只含变量名和占位值；PingCode ID：`PENDING`；已完成：文档与示例配置检查通过）
- [x] 5.2 完成 FR-011 测试矩阵与 CI 整体验收：鉴权、令牌刷新、分页、异常状态、超时/取消、只读拒绝、白名单拒绝、MCP 协议；修复并运行格式、vet、build、test（关联 FR-002、FR-003、FR-010、FR-011；Spec：五项 capability；验收方式：`gofmt` 检查、`go vet ./...`、`go build ./...`、`go test ./...` 全部通过且不依赖真实 PingCode 凭据；PingCode ID：`PENDING`；已完成：实现覆盖全部矩阵项，本地与 CI 命令（含 `-race`）通过）
- [ ] 5.3 在负责人授权的隔离测试项目，用稳定版 MCP 创建、读取、更新并回读验证一个工作项；记录脱敏证据（关联 FR-012；Spec：`mcp-workitem-tools`、`write-safety`；验收方式：独立 `INTEGRATION` 记录授权范围、操作时间、工作项 ID 与回读结果；PingCode ID：`PENDING`；当前状态：`INTEGRATION-PENDING`，依赖真实凭据、隔离测试项目与明确写入授权；公开仓库不保存真实标识符或运行时间）

## Workflow follow-up

- 负责人已批准本设计（第一道 Gate 通过，对应 CODEX_PROMPT Step 3→4）；实现按本清单顺序推进，任务勾选随验证更新。
- FR-012 只可在获准的隔离测试项目和明确写入授权下执行；未获得授权时保持未完成。
- 全部实现验收完成后，再由负责人决定是否归档本 change。
