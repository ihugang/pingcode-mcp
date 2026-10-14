# Design

> 状态：负责人已在 2026-10-14 明确批准按本 change 执行；实现后仍需负责人独立验收。

## Context

- 现有 PingCode 服务是本地 MCP stdio 程序；它的启动配置和工具行为见 README 与 `docs/api-contract.md`。
- 负责人批准本 change 时确认：PRD FR-003 覆盖 Codex 与其他支持 MCP stdio 的 IDE，以及配套跨 IDE Skill/接入说明。PRD FR-003 已同步更新；PingCode 需求 ID 在真实需求建立前保持 `PENDING`。
- Kiro IDE 当前支持 Agent Skills 标准、workspace/global 技能目录，并可从 GitHub 子目录或 `SKILL.md` URL 导入。Codex、Claude Code、Cursor 与 VS Code/GitHub Copilot 使用兼容技能格式，但安装位置与 GitHub 导入方式各不相同。实现时须再次核对官方文档，不能将格式兼容等同于安装 UI 一致。

## Goals

- 提供一份 IDE 无关的 Agent Skill 主说明，并按需读取各 IDE 的安装配置参考。
- 用户可以清楚地区分两个操作：把 Skill 安装到 Agent，以及把 PingCode MCP Server 注册到 IDE。
- GitHub 上的 Skill 目录可被 Kiro 导入，也可供其他支持 Agent Skills 的客户端安装。
- 不更改当前服务的只读默认值、白名单要求、凭据来源或工具集合。

## Proposed Structure

```text
skills/pingcode-mcp-setup/
├── SKILL.md
├── scripts/
│   └── install.py
└── references/
    └── ide-setup.md
.github/workflows/release.yml
```

主文件保留任务识别、信息收集、安全约束和安装器入口；具体 IDE 配置格式放进引用文档。`install.py` 负责本机安装和 MCP 配置，不实现 PingCode API 调用，不读取或保存 Secret。除非负责人要求，否则不重复维护一份与 README 完全相同的安装手册。

## Installer Contract

- 支持 `--client` 选择单个目标 IDE、`--scope` 选择 user/workspace、`--workspace` 指定工作区根目录、`--version` 固定版本、`--dry-run` 预览、`--replace` 替换冲突项、`--yes` 用于显式非交互确认，以及 `--help` 查看选项。若参数不足且运行在交互终端，可询问目标；非交互执行时必须给出明确错误和下一步命令。一次只配置一个 IDE，避免意外修改多个客户端。
- 正常安装从公开 GitHub Releases 下载预编译资产，不要求用户安装 Go 或 Git。Python 安装器使用标准库且要求 Python 3.11+；Windows 提供 `py -3` 调用方式。脚本支持 `--version` 固定版本；未指定时选择最新正式版本。若没有兼容资产或尚无正式 release，停止并说明原因；只有用户显式指定 `--build-from-source` 时才要求 Go 1.25+ 并从指定 tag 的 GitHub 源码归档本机构建。
- Release workflow 仅由 `vX.Y.Z` 版本 tag 触发，并依赖 CI 格式检查、`go vet ./...`、`go build ./...`、`go test ./...` 成功。交叉编译 `darwin/{amd64,arm64}`、`linux/{amd64,arm64}`、`windows/{amd64,arm64}`；分别生成 tar.gz 或 zip、版本元数据和 `SHA256SUMS`。
- 安装器下载与所选 OS/架构匹配的 release archive，校验 `SHA256SUMS` 后才解包；仅提取预期二进制文件名，并拒绝 archive 路径穿越。GitHub Actions 为 release 二进制生成 provenance attestation；安装器支持用 GitHub CLI 验证 attestation，若 CLI 不存在则明确报告跳过状态并仍验证 SHA-256。
- 把二进制安装到用户专属目录；配置写入所选 IDE 官方支持的用户级或工作区级位置。工作区级配置需要明确工作区根目录。
- Kiro、Cursor、VS Code/GitHub Copilot 的 JSON 配置按各自 schema 更新；Codex、Claude Code 等优先调用官方 MCP CLI 命令。每种实现必须尽量原子写入并保留无关配置。
- 在任何下载、解包或配置写入前完成平台兼容、版本、目标路径、现有同名项和预期变更检查，并展示计划。仅在用户确认后下载/校验/安装二进制、写入配置；`--dry-run` 不联网、不解包、不写文件，仅展示本机平台、目标目录和配置计划。源码构建模式还须先明确告知 Go 前置条件。
- 对已存在的 `pingcode` 服务，若配置一致则报告已安装；若配置不同则停止并展示脱敏差异，要求用户确认替换。未经确认不得替换。配置更新采用临时文件和原子替换，保留既有文件权限和无关服务；不另建可能复制其他服务 Secret 的备份文件。
- 不把 Client ID、Client Secret、Token 或 Authorization Header 写入配置。只读模式由服务本身默认开启；项目白名单和凭据由用户通过本机环境配置，空白名单不得被脚本自动填充。
- 安装后执行本地 MCP `initialize` 与工具发现验证，不调用 PingCode API。若缺少 Go、Git、Python 或客户端 CLI，明确失败原因并提供手动配置指引，不尝试静默安装这些依赖。

脚本优先使用 Python 标准库完成平台检测、HTTPS 下载、SHA-256 校验、安全解包、子进程调用、JSON 更新和原子替换，不引入第三方 Python 包。Codex TOML 配置与客户端 CLI 的行为必须在实现时依据当前官方文档核对；不得用脆弱的字符串替换破坏已有配置。

## Behavior

1. 识别用户要安装的是 Agent Skill、PingCode MCP Server，还是两者都要；用户请求安装服务时可运行安装器。
2. 对 MCP Server 部署，先识别操作系统、架构、单个 IDE/Agent、user/workspace 范围和源码来源；参数不完整时询问，不猜目标配置路径。
3. 安装器先只读检查平台、可用 release 版本、校验能力、目标配置和同名冲突，展示完整计划；用户确认后才下载并验证 release、安装二进制、写入配置。dry-run 和用户取消不改变系统状态。
4. 安装时不覆盖同名冲突项，不更改其他 MCP server。
5. 所有示例默认 `PINGCODE_READ_ONLY=true`；项目白名单为空时说明项目调用不会被授权。不得预批准全部工具。
6. Secret 只能由用户通过环境变量或本机安全凭据方式提供。不得要求用户将 Secret 粘贴到对话，也不得写入 Git、Skill、IDE 共享配置、工具参数或日志。
7. 验证仅覆盖本地构建、MCP initialize 和 tools/list；不调用 PingCode API。除非另有真实授权，不执行远端读写；真实 API 验证必须标记为 `INTEGRATION`。
8. 文件系统安装行为仅针对用户明确选择的目标和范围。报告构建结果、配置文件路径、预览/变更内容与验证结果。

## Initial Client Coverage

- Kiro IDE：workspace/global skill import 和本地 stdio MCP 配置。
- Codex：用户级或仓库级 skill 目录及 MCP 配置。
- Claude Code：project/user skill 目录及 MCP 配置。
- Cursor：workspace skill 和 MCP 配置。
- VS Code / GitHub Copilot：workspace skills 目录和 MCP 配置。

兼容性必须依据 Agent Skills 格式与客户端对路径、导入方式和 MCP transport 的实现分别判断。Skill 不提供所有 IDE 完全相同的一键安装体验。

## Risks / Trade-offs

- IDE 文档与导入 UI 会变化 → 参考文档优先链接官方资料，并在 Skill 实际使用或发布前核对当前文档；避免把易变化信息放进 Skill 主文件。
- 一个 Skill 可能被误认为会自动安装 MCP Server → 主文件和 README 明确区分 Skill 导入与 MCP 配置两个步骤。
- 跨 IDE 配置容易覆盖用户已有 MCP 项 → 只合并目标 server 项，保留 JSON 中的其他 server 与设置，并展示变更摘要。
- 已检查 `openspec` 目录但本环境未安装 OpenSpec CLI → 本草案按仓库现有文件结构撰写；后续通过可用 CLI 或官方校验方式检查结构。

## Rollout / Rollback

已按负责人批准开始实现 release workflow、安装脚本及官方文档适配；验证交叉编译、校验失败、错误平台、配置冲突、取消、超时和配置解析错误；再在代表性 IDE 中检查工具发现。负责人独立验收后决定是否提交、推送或创建首个正式 release。安装器不提供删除行为。
