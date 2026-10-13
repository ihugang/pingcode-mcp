# PingCode MCP Server

PingCode MCP Server 是一个基于官方 Go MCP SDK 的本地 stdio 服务，为 MCP 客户端提供受控的 PingCode 项目与工作项访问能力。当前实现默认只读，项目定向读取和写入都受 `PINGCODE_ALLOWED_PROJECT_IDS` 白名单限制；stdout 只承载 MCP 协议，日志写入 stderr。

> **发布状态：** 本项目以 [MIT License](LICENSE) 发布，公开仓库地址为 <https://github.com/ihugang/pingcode-mcp>。当前没有正式版本或 GitHub Release；FR-012 真实租户验收仍为 `INTEGRATION-PENDING`。

## 支持范围

当前 MVP 只提供本地 MCP stdio 进程和以下六个工具：

- `pingcode_list_projects`
- `pingcode_get_project`
- `pingcode_list_work_items`
- `pingcode_get_work_item`
- `pingcode_create_work_item`
- `pingcode_update_work_item`

当前不提供 Wiki、Authorization Code、Streamable HTTP、共享多租户 HTTP 服务、删除工具或批量写入工具。这些属于 PRD 中的后续范围或明确非目标，不应通过本次文档发布默认承诺。

## 快速开始

### 前置条件

- Go 1.25 或更新版本。`go.mod` 声明 Go 1.25；设置 `GOTOOLCHAIN=auto` 时，Go 可自动获取所需工具链。
- 一个支持 MCP stdio 的客户端。客户端的配置文件位置和 JSON 外壳会因客户端及版本而不同，请以客户端文档为准。
- 真实 PingCode 调用需要已获授权的 Client ID、Client Secret 和目标项目权限；离线测试不需要这些凭据。

### 构建并验证

```sh
gofmt -l .                 # 应无输出
go vet ./...
go build ./...
go test ./...

mkdir -p release/darwin
go build -o release/darwin/pingcode-mcp ./cmd/pingcode-mcp
```

发行构建产物应放在按操作系统划分的 `release/` 子目录中；例如 `release/darwin/`、`release/linux/` 或 `release/windows/`（Windows 使用 `.exe` 后缀）。这些目录和根目录遗留的 `pingcode-mcp` 构建产物均由 `.gitignore` 忽略，不应提交到 Git。上述测试使用替身、HTTP mock 和 MCP 协议 harness，不连接真实 PingCode，也不需要本机凭据。

### 配置并启动

程序只读取进程环境变量，**不会自动解析 `.env`、`.pingcode.env` 或其他 dotenv 文件**。如使用被 `.gitignore` 忽略的本地文件，请自行将其中的 `PINGCODE_*` 赋值导入当前 shell；不要覆盖已有本地凭据文件，也不要把文件内容复制到 Issue、日志、截图或对话中：

```sh
# .pingcode.env 只能包含本机环境变量赋值，键名必须是 PINGCODE_*。
# 不要在公开仓库中创建、提交或展示真实值。
set -a
. ./.pingcode.env
set +a
./release/darwin/pingcode-mcp
```

服务可以在没有凭据时启动；令牌会在第一次真实 API 调用时惰性获取。无凭据启动不等于已经完成 PingCode 鉴权。

## 配置

凭据只能通过进程环境变量提供。Client ID 和 Client Secret 必须同时设置，或同时不设置。

| 变量 | 默认值 / 校验 | 用途 |
| --- | --- | --- |
| `PINGCODE_CLIENT_ID` | 未设置时允许离线启动；真实 API 调用前必填 | OAuth Client ID |
| `PINGCODE_CLIENT_SECRET` | 未设置时允许离线启动；真实 API 调用前必填 | OAuth Client Secret |
| `PINGCODE_BASE_URL` | `https://open.pingcode.com`；必须是无 query/fragment 的绝对 HTTP(S) URL | PingCode REST 根地址 |
| `PINGCODE_READ_ONLY` | 缺省或空值为 `true`；只有显式设为 `false` 才允许继续检查写入授权 | 全局写入开关 |
| `PINGCODE_ALLOWED_PROJECT_IDS` | 逗号分隔；空值为空白名单 | 项目授权范围；空白名单不授权项目定向读写 |
| `PINGCODE_HTTP_TIMEOUT` | `30s`；必须为正 duration | HTTP 请求超时 |
| `PINGCODE_LOG_LEVEL` | `info`；可用 `debug`、`info`、`warn`、`error` | stderr 日志等级 |

`PINGCODE_ALLOWED_PROJECT_IDS` 同时限定项目定向读取与写入。写操作还必须通过 `PINGCODE_READ_ONLY=false` 这一全局开关；真实写入只应在负责人批准的隔离测试项目中进行，完成后立即恢复只读。

## MCP 客户端接入

先构建程序，并将客户端配置中的 `command` 改为二进制的**绝对路径**。下面是常见 MCP 配置形态；不同客户端可能使用不同的顶层字段或配置文件位置，请按客户端文档调整。值全部是占位符，不要将真实 Secret 写进仓库或工具参数：

```json
{
  "mcpServers": {
    "pingcode": {
      "command": "/absolute/path/to/release/darwin/pingcode-mcp",
      "env": {
        "PINGCODE_CLIENT_ID": "replace-with-client-id",
        "PINGCODE_CLIENT_SECRET": "replace-with-client-secret",
        "PINGCODE_ALLOWED_PROJECT_IDS": "approved-project-id",
        "PINGCODE_READ_ONLY": "true"
      }
    }
  }
}
```

接入时请注意：

1. 客户端进程必须能读取配置中的环境变量；如果客户端不支持安全的环境变量注入，不要把 Secret 改放到 MCP 工具参数中，应使用客户端或操作系统提供的 Secret 管理方式。
2. `PINGCODE_READ_ONLY` 应保持为 `true`。白名单只填写负责人批准的项目**内部 ID**，不要使用项目展示名称或 identifier。
3. 启动后先执行 MCP `initialize` 和 `tools/list`，确认能发现上述六个工具；服务日志应出现在 stderr，stdout 不应混入普通日志。
4. 真实只读调用前确认 Client ID、Client Secret、应用 scope 和项目权限均已获授权。无凭据或权限不足时，预期看到结构化的 `AUTHENTICATION` 或 `PERMISSION` 错误，而不是把凭据放进请求参数。
5. 只有在明确授权的隔离测试项目中，才可临时设置 `PINGCODE_READ_ONLY=false` 并执行单项创建/更新；完成后停止进程或恢复为 `true`。

### 最小接入检查

- **离线检查：** 在无 `PINGCODE_*` 凭据的环境运行 `go test ./...`，确认本地构建和测试通过。
- **MCP 检查：** 客户端完成 `initialize`、`tools/list`，且工具列表只包含当前支持范围。
- **真实只读检查（`INTEGRATION`）：** 在授权环境中设置只读和项目白名单，调用 `pingcode_list_projects` 或 `pingcode_get_project`，记录脱敏结果。没有真实授权时不要声称完成联调。

## 工具

| 名称 | 用途 | 必填参数 | 要点 |
| --- | --- | --- | --- |
| `pingcode_list_projects` | 列出项目白名单内的项目 | 无 | 支持过滤和分页；空白名单返回空列表 |
| `pingcode_get_project` | 获取项目详情 | `project_id` | `project_id` 必须精确命中白名单 |
| `pingcode_list_work_items` | 按条件分页查询工作项 | `project_id` | 项目必须命中白名单 |
| `pingcode_get_work_item` | 获取工作项详情 | `workitem_id`、`project_id` | 服务会核对工作项实际所属项目 |
| `pingcode_create_work_item` | 在白名单项目创建单个工作项 | `project_id`、`type_id`、`title` | 仅单项创建；类型映射仍以目标租户确认结果为准 |
| `pingcode_update_work_item` | 更新白名单项目中的单个工作项 | `workitem_id`、`project_id`，以及至少一个可更新字段 | 只发送调用方提供的更新字段 |

列表工具支持 `page_size`（默认 `30`，最大 `100`）和从 `0` 开始的 `page_index`。完整字段与输入限制以 MCP Schema 和 [docs/api-contract.md](docs/api-contract.md) 为准。

## 结构化错误

工具错误以 `{"error":{"type":"...","message":"..."}}` 形式返回。`message` 用于安全诊断，不包含凭据。

| `error.type` | 常见触发条件 | 处置建议 |
| --- | --- | --- |
| `VALIDATION` | 缺少必填字段、分页参数无效、更新未提供字段 | 检查工具参数和分页范围后重试 |
| `AUTHENTICATION` | 未配置凭据、Client Credentials 鉴权失败或令牌无效 | 核对本地环境和应用授权状态；不要把凭据发到对话或日志 |
| `PERMISSION` | PingCode 拒绝当前资源或操作 | 核对应用 scope、账号和目标项目权限 |
| `NOT_FOUND` | 指定项目或工作项不存在 | 核对资源 ID 与 `project_id` |
| `RATE_LIMITED` | PingCode 返回限流响应 | 按服务端提示等待，避免立即重复请求 |
| `UPSTREAM` | PingCode 服务错误或未分类上游错误 | 稍后重试并检查脱敏摘要 |
| `TIMEOUT` | HTTP 请求超时 | 检查网络，必要时调整 `PINGCODE_HTTP_TIMEOUT` |
| `CANCELLED` | 请求被调用方或进程取消 | 确认客户端请求仍有效后重试 |
| `READ_ONLY` | 当前配置为只读，写操作被拒绝 | 先确认负责人授权、隔离项目和白名单，再决定是否临时关闭只读 |
| `PROJECT_NOT_ALLOWED` | 项目 ID 不在白名单中 | 核对目标项目内部 ID；只有获批项目可加入白名单 |

## 安全边界

- 默认只读；任何写操作都要求显式 `PINGCODE_READ_ONLY=false`，且项目 ID 精确命中 `PINGCODE_ALLOWED_PROJECT_IDS`。
- 凭据、令牌、Authorization Header 和敏感工作项正文不会写入日志或错误消息。
- 不提供删除或批量写入工具；本地白名单是额外保护，不替代 PingCode 服务端权限控制。
- 发现凭据疑似泄露时，应立即停止相关本地进程并按 [SECURITY.md](SECURITY.md) 的流程私下报告和轮换；不要把值粘贴到公开渠道。

## 验证和集成状态

离线验证包括格式检查、`go vet ./...`、`go build ./...`、`go test ./...`，均不需要真实 PingCode 凭据。授权的真实租户验证若存在，必须单独标记为 `INTEGRATION` 并保存在不进入公开仓库的受控记录中；本公开版本的状态为 `INTEGRATION-PENDING`。契约证据和仍待确认的行为见 [docs/api-contract.md](docs/api-contract.md)。真实验证记录不能推广为所有租户或所有字段均已保证，尤其是令牌 `expires_in` 语义、实际 scope、限流阈值和部分可选字段仍应以文档中的状态标记为准。

## 开发与协作

- 开发环境、测试门槛、Spec/FR 追踪和 PR 要求见 [CONTRIBUTING.md](CONTRIBUTING.md)。
- 行为规范见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。
- 安全漏洞报告流程见 [SECURITY.md](SECURITY.md)，通过 GitHub Security Advisories 私下报告。
- 当前变更记录见 [CHANGELOG.md](CHANGELOG.md)。
- 本项目按 [MIT License](LICENSE) 发布。

## 相关文档

- [PRD](docs/PRD.md)
- [PingCode API 契约](docs/api-contract.md)
- [FR-012 自举指南](docs/BOOTSTRAP.md)
- [.env.example](.env.example)
