# Cross-IDE PingCode MCP Setup Skill

## ADDED Requirements

### Requirement: Portable Agent Skill Package

项目 SHALL 提供一个 PingCode MCP 安装 Skill 目录，包含有效的 Agent Skills `SKILL.md`、可执行本地安装器及按需读取的客户端参考。Skill MUST 区分导入 Skill 与安装/注册 MCP Server。

README SHALL 提供可复制给 IDE Agent 的 Skill 导入提示词和 MCP Server 安装提示词。两段提示词 MUST 明确各自操作范围；MCP Server 安装提示词 MUST 要求先预览计划并等待用户明确确认，且保留代理设置、只读默认值和凭据安全边界。

#### Scenario: 将 Skill 导入 Kiro

- **GIVEN** GitHub 子目录中包含 `SKILL.md`、安装器和 references
- **WHEN** 用户通过 Kiro 的 Agent Steering & Skills 导入该目录
- **THEN** Skill 可在 Kiro IDE 中发现
- **AND** Skill 说明 MCP Server 注册是独立步骤
- **AND** Skill 可调用其随附安装器部署命令并配置 Kiro

#### Scenario: 安装到其他兼容客户端

- **GIVEN** 客户端支持 Agent Skills 标准
- **WHEN** 用户按该客户端支持的 user 或 workspace 目录安装此 Skill
- **THEN** 客户端可以发现同一份核心指令
- **AND** Skill 参考分别说明该客户端的 MCP 配置差异

#### Scenario: 通过 README 提示词安装

- **GIVEN** 用户将 README 中对应的提示词提供给兼容的 IDE Agent
- **WHEN** 用户请求导入 Skill 或安装 MCP Server
- **THEN** Skill 导入提示词只安装 Skill，不触发 MCP Server 下载或配置
- **AND** MCP Server 安装提示词先运行 `--dry-run` 并展示目标与变更计划，等待用户明确确认后才执行下载和配置写入
- **AND** 下载沿用运行环境代理设置，配置保持 `PINGCODE_READ_ONLY=true`，Agent 不索取或保存 PingCode 凭据

### Requirement: Client-Specific MCP Setup Guidance

Skill SHALL 提供带官方来源链接的 Kiro IDE、Codex、Claude Code、Cursor、VS Code/GitHub Copilot 指引。指引 MUST 区分通用 Agent Skills 格式与产品专属导入机制。安装器 SHALL 支持选择客户端和 user/workspace 范围、预览变更，并保留无关配置项。

#### Scenario: 配置本地 stdio 服务

- **GIVEN** 用户选择一个支持的客户端和当前平台
- **WHEN** Skill 或安装器生成 MCP 配置
- **THEN** 配置位置与 schema 符合该客户端当前文档
- **AND** 保留配置中的其他 MCP Server 项

#### Scenario: 预览并确认安装

- **GIVEN** 用户请求安装 PingCode MCP 命令
- **WHEN** 安装器解析目标二进制位置与客户端配置
- **THEN** 安装器在写入前展示文件和配置变更计划
- **AND** dry-run 或用户取消不会更改系统状态
- **AND** 不会未经明确替换确认覆盖同名 `pingcode` 配置

### Requirement: Versioned Multi-Platform Release Installation

项目 SHALL 通过 GitHub Releases 发布 macOS、Linux、Windows 的 amd64 和 arm64 预编译 PingCode MCP 程序。每个 release SHALL 提供资产 SHA-256 manifest 和构建来源 attestation。安装器 SHALL 选择匹配当前平台的资产，在解包前校验 SHA-256，并允许固定 release 版本。源码构建 MUST 由用户显式选择。

#### Scenario: 从匹配的 Release 安装

- **GIVEN** 稳定版本包含匹配用户操作系统与架构的资产
- **WHEN** 用户确认安装计划
- **THEN** 安装器下载所选版本及 SHA256SUMS
- **AND** 在解包前验证该资产的摘要
- **AND** 安装正确程序且不要求用户安装 Go
- **AND** 本机存在 GitHub CLI 时验证 attestation 的仓库与 workflow 身份

#### Scenario: 无兼容 Release

- **GIVEN** 尚无稳定 Release 或无匹配平台资产
- **WHEN** 安装器检查所选版本
- **THEN** 不修改本机文件或 IDE 配置
- **AND** 说明原因并给出需显式选择且要求 Go 1.25+ 的源码构建选项

#### Scenario: 拒绝损坏或不安全的资产

- **GIVEN** 下载资产的 SHA-256 不匹配、压缩包含路径穿越或额外非预期文件，或可用的 attestation 验证失败
- **WHEN** 安装器校验 release 资产
- **THEN** 安装器拒绝该资产
- **AND** 保留原二进制与 MCP 配置

### Requirement: Preserve PingCode Safety Defaults

Skill SHALL 保持服务的只读默认值、项目 ID 白名单、仅从环境变量提供凭据、无删除工具和无批量写入边界。Skill MUST NOT 包含真实凭据，也 MUST NOT 建议把凭据存入仓库文件、共享 Skill 文件、MCP 工具参数或日志。

#### Scenario: 准备默认安全配置

- **GIVEN** 用户未明确授权写操作
- **WHEN** Skill 生成或说明 MCP 配置
- **THEN** 配置保持 `PINGCODE_READ_ONLY=true`
- **AND** 空白项目白名单不会授予项目访问权限
- **AND** 不把任何工具加入自动批准列表

#### Scenario: 无 PingCode 凭据安装

- **GIVEN** 用户还没有配置 PingCode 凭据
- **WHEN** 安装器安装并注册本地 stdio 程序
- **THEN** 安装器不索取或写入 Client ID、Client Secret
- **AND** 可以在不调用 PingCode 的前提下验证本地 MCP 初始化和工具发现

#### Scenario: PingCode 授权不可用

- **GIVEN** 用户没有已授权凭据或项目范围
- **WHEN** Skill 帮助用户检查安装
- **THEN** 验证范围仅限本地程序与 MCP `initialize`/`tools/list`
- **AND** 不声称真实 PingCode 联调完成，也不执行远端写操作
- **AND** 未来真实租户 API 验证须标记为 `INTEGRATION`
