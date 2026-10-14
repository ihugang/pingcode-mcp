# IDE 与 MCP 客户端参考

查询日期：2026-10-14。产品入口和目录可能随版本变化；执行前以所链接官方文档为准。Agent Skills 格式相同，不代表各 IDE 使用同一导入入口或 MCP 配置 schema。

## 安装 Skill 目录

请将整个 `pingcode-mcp-setup` 目录复制到对应目录，保留其脚本和 references 子目录。

| 客户端 | workspace Skill | user Skill | 官方说明 |
| --- | --- | --- | --- |
| Kiro IDE | `.kiro/skills/pingcode-mcp-setup/` | `~/.kiro/skills/pingcode-mcp-setup/` | [Kiro Agent Skills](https://kiro.dev/docs/skills/) 支持从公开 GitHub 子目录导入 |
| Codex | `.agents/skills/pingcode-mcp-setup/` | `~/.agents/skills/pingcode-mcp-setup/` | [Codex Skills](https://developers.openai.com/codex/skills/) |
| Claude Code | `.claude/skills/pingcode-mcp-setup/` | `~/.claude/skills/pingcode-mcp-setup/` | [Claude Code Skills](https://code.claude.com/docs/en/skills) |
| Cursor | `.agents/skills/pingcode-mcp-setup/` 或 `.cursor/skills/pingcode-mcp-setup/` | `~/.agents/skills/pingcode-mcp-setup/` 或 `~/.cursor/skills/pingcode-mcp-setup/` | [Cursor Agent Skills](https://cursor.com/docs/skills) |
| VS Code / GitHub Copilot | `.github/skills/pingcode-mcp-setup/`、`.agents/skills/pingcode-mcp-setup/` 或 `.claude/skills/pingcode-mcp-setup/` | `~/.copilot/skills/`、`~/.agents/skills/` 或 `~/.claude/skills/` | [VS Code Agent Skills](https://code.visualstudio.com/docs/agent-customization/agent-skills) |

Kiro IDE 的 GitHub 导入 URL 必须指向仓库子目录或 `SKILL.md`，不能只填仓库根 URL。导入 Skill 后还需运行安装脚本，才能配置 MCP 服务。

## 安装 MCP 程序

安装器支持以下客户端和范围：

```sh
python3 <skill-directory>/scripts/install.py --client kiro --scope user --dry-run
python3 <skill-directory>/scripts/install.py --client kiro --scope user
python3 <skill-directory>/scripts/install.py --client vscode --scope workspace --workspace /absolute/path/to/project
```

Windows 使用 `py -3`。Python 3.11+ 是安装器前置条件；常规预编译安装不需要 Go 或 Git。非交互运行需加 `--yes`。替换同名服务或已安装程序时还需加 `--replace`。如果使用 `--build-from-source`，本机 Go 必须为 1.25+。

默认安装目标和配置格式：

| 客户端 | user 配置 | workspace 配置 | 格式 |
| --- | --- | --- | --- |
| Kiro | `~/.kiro/settings/mcp.json` | `.kiro/settings/mcp.json` | JSON，`mcpServers` |
| Codex | `~/.codex/config.toml` | `.codex/config.toml` | TOML，`[mcp_servers.pingcode]`；项目设置仅对受信任项目加载 |
| Claude Code | Claude Code CLI 的 user scope | Claude Code CLI 的 project scope，写入 `.mcp.json` | `claude mcp add-json` |
| Cursor | `~/.cursor/mcp.json` | `.cursor/mcp.json` | JSON，`mcpServers`，stdio 项需要 `type: "stdio"` |
| VS Code / Copilot | `~/.copilot/mcp-config.json` | `.mcp.json` | portable JSON，`mcpServers` |

下载 GitHub Release 时，安装器通过 Python 标准库读取当前进程的 `HTTP_PROXY`/`HTTPS_PROXY`（也接受小写形式）；对应协议未设置时会尝试 `ALL_PROXY`，并遵守 `NO_PROXY`。当前只支持 HTTP(S) 代理；如果只配置 SOCKS 代理且目标不在 `NO_PROXY` 中，安装器会停止并要求提供 HTTP(S) 代理。请在启动安装器的终端或 IDE 环境中设置代理；安装器不会显示代理值或写入客户端配置。

workspace 配置可能进入 Git。安装器仅添加 `PINGCODE_READ_ONLY=true`，不得添加真实凭据。VS Code 同样支持 `.vscode/mcp.json` 的 VS Code 专用格式；本安装器使用 portable `.mcp.json` 以适配 Agent Host。

安装器将程序放到当前用户目录，并在配置中使用绝对程序路径：

- macOS/Linux：`~/.local/bin/pingcode-mcp`
- Windows：`%LOCALAPPDATA%\Programs\PingCodeMCP\pingcode-mcp.exe`

## 客户端官方 MCP 文档

- [Kiro MCP 配置](https://kiro.dev/docs/mcp/configuration/)：workspace 使用 `.kiro/settings/mcp.json`，user 使用 `~/.kiro/settings/mcp.json`，格式为 `mcpServers`。
- [Codex MCP](https://developers.openai.com/codex/mcp/)：user 配置为 `~/.codex/config.toml`；项目级 MCP 放在受信任项目的 `.codex/config.toml`。
- [Claude Code MCP](https://code.claude.com/docs/en/mcp)：通过 `claude mcp add-json` 注册，user scope 对所有项目可用，project scope 写入 workspace 的 `.mcp.json`。
- [Cursor MCP](https://cursor.com/docs/mcp)：user 使用 `~/.cursor/mcp.json`，project 使用 `.cursor/mcp.json`；stdio 项声明 `type: "stdio"`。
- [VS Code / Copilot MCP](https://code.visualstudio.com/docs/agent-customization/mcp-servers)：workspace 支持 `.mcp.json`（portable `mcpServers`）或 `.vscode/mcp.json`（VS Code 格式）；user portable 路径为 `~/.copilot/mcp-config.json`。

## Release 资产与校验

正式 tag `vMAJOR.MINOR.PATCH` 触发 GitHub Actions Release workflow。每个 release 为 macOS、Linux、Windows 的 amd64 和 arm64 发布 tar.gz/zip 资产、`VERSION.txt`、`SHA256SUMS`，并为 release 资产生成 GitHub provenance attestation。

安装器在解包前验证 `SHA256SUMS`，只读取预期二进制，拒绝路径穿越和额外文件。若安装机上已有 GitHub CLI，会验证 attestation 的仓库和 signer workflow；没有 `gh` 时仅能验证从同一 GitHub Release 下载的 SHA-256 清单，安装器会明确提示这不是独立的签名来源验证。[GitHub artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)；[验证命令](https://cli.github.com/manual/gh_attestation_verify)。

如果 GitHub Release 尚不存在或没有匹配资产，安装器不做更改。源码构建只在用户显式指定 `--build-from-source` 后执行。

## 验证与凭据

安装器本地启动程序，只发送 MCP `initialize`、`notifications/initialized` 和 `tools/list`。验证期间会移除当前进程的 `PINGCODE_*` 环境变量，并将 read-only 强制为 `true`、项目白名单置空；不会连接 PingCode。

需要访问真实项目时，用户要自行通过操作系统或 IDE 支持的本机安全机制向 MCP 进程注入凭据和经批准的项目 ID。不要把这些值写入被提交的 `.mcp.json`、Codex TOML、skill 文件或日志。没有凭据/授权时只报告本地 MCP 验证结果；真实服务端调用必须标为 `INTEGRATION`。
