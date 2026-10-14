# Proposal

## Why

当前 README 已说明 PingCode MCP Server 的环境变量、构建和通用 stdio 接入，但 IDE 配置入口和文件格式存在差异。用户需要在 Kiro 等 Agent Skills 兼容 IDE 中重复查找并转换配置，容易把 MCP Server 注册与 Skill 安装混为一谈，也可能将凭据写进被提交的配置文件。

本变更将可复用的跨 IDE 安装流程整理为符合 Agent Skills 开放规范的目录，发布在现有 GitHub 仓库，使用户可从 Kiro 导入，并安装到其他兼容客户端。

## What Changes

- 新增 `skills/pingcode-mcp-setup/SKILL.md` 和可执行安装脚本。GitHub Actions 在版本 tag 上构建各平台 release 二进制；脚本按用户系统下载、校验并安装单个 PingCode MCP 命令，再注册到用户选择的 IDE。
- 为 macOS、Linux 和 Windows 的 amd64/arm64 目标发布预编译资产及 SHA-256 校验文件；安装脚本支持指定版本，并只在用户明确选择源码构建时要求本机 Go 工具链。
- 添加按需加载的 IDE 配置适配和官方资料链接，初始覆盖 Kiro IDE、Codex、Claude Code、Cursor、VS Code/GitHub Copilot；文档路径和配置格式以各产品官方资料为准。
- 安装脚本支持预览下载、校验和配置变更，保留目标配置中其他服务，避免覆盖同名且内容不同的服务，并在安装完成后验证 MCP 初始化和工具发现。
- 保留现有服务的安全边界：凭据不进入 Git 或 MCP JSON；只读默认开启；项目读取/写入受项目 ID 白名单约束；写操作须有明确授权。
- 补充 README 中的 GitHub 安装入口和维护说明。

## Traceability

- PRD：FR-003（MCP stdio 与跨 IDE Skill/接入说明；负责人已确认扩展范围并同步 PRD）、FR-010（凭据、只读及项目白名单安全要求）、FR-011（发布构建与离线验证）。
- OpenSpec：`pingcode-mcp-setup-skill` change；具体工作见 `tasks.md`。
- PingCode 需求 ID：`PENDING`。

## Out of Scope

- 不修改 PingCode MCP Server 的协议、工具、传输方式或 PingCode API 行为；安装器只负责本地构建与 MCP 客户端注册。
- 不把 Client ID、Client Secret、Token、Authorization Header 或本机凭据文件纳入技能、示例、快照或 Git。
- 不自动执行真实 PingCode API 调用或写操作，也不宣称完成真实租户联调。
- 不承诺不支持 Agent Skills 规范的 IDE 具有原生 Skill 激活能力；可提供手动复制或等价工作流说明。

## Impact

- 新增一个 Agent Skills 标准技能目录、跨平台 release workflow、安装脚本和一个 OpenSpec capability delta。
- README 增加 Skill 的定位、GitHub 导入方式及支持范围。
- 不新增 PingCode Server 运行时依赖或 Go 业务代码；安装脚本只使用随操作系统可用的标准工具，并明确列出源码构建所需的 Go 版本。
