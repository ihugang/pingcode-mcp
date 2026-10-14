---
name: pingcode-mcp-setup
description: Install the PingCode MCP stdio server and configure it for Kiro or another compatible IDE. Use when importing this skill, installing or updating the PingCode MCP command, configuring an MCP client, or troubleshooting local tool discovery.
metadata:
  author: ihugang
  repository: https://github.com/ihugang/pingcode-mcp
---

# PingCode MCP 安装配置

本 Skill 说明如何安装本 Skill 本身，以及如何单独安装并注册 PingCode MCP 服务。导入 Skill 不会自动下载或启动服务。

## 工作流程

1. 确认用户要做的是导入 Skill、安装/更新 MCP 服务，还是配置客户端。不要把这几个步骤当成同一件事。
2. 安装 MCP 服务前确认目标客户端（Kiro、Codex、Claude Code、Cursor 或 VS Code/GitHub Copilot）和范围（user 或 workspace）。workspace 范围要求用户明确给出已存在的工作区目录。
3. 如需查看客户端路径、Skill 目录和官方 MCP 配置差异，只读取 [references/ide-setup.md](references/ide-setup.md) 中对应客户端的小节。
4. 使用本 Skill 的 `scripts/install.py`。先运行 `--dry-run` 查看目标，再运行正式安装命令。脚本会展示计划并在写配置前要求确认；非交互运行需要 `--yes`。已有同名配置或安装程序需要用户明确加 `--replace`。

```sh
python3 <skill-directory>/scripts/install.py --client kiro --scope user --dry-run
python3 <skill-directory>/scripts/install.py --client kiro --scope user
```

Windows 可使用 `py -3` 替代 `python3`。workspace 级安装要增加 `--workspace <现有工作区绝对路径>`。版本固定示例：`--version v1.2.3`。正常安装从 GitHub Releases 下载预编译程序；只有用户明确要求源码构建时才使用 `--build-from-source`，该方式要求 Go 1.25+。

安装器下载 GitHub Release 时会读取当前进程的代理设置（`HTTP_PROXY`/`HTTPS_PROXY` 及其小写形式；未设置对应协议代理时也会尝试 `ALL_PROXY`），并遵守 `NO_PROXY`。当前只支持 HTTP(S) 代理；若仅配置了 SOCKS 代理且目标不在 `NO_PROXY` 中，安装器会安全停止。请在运行安装器的终端或 IDE 环境中配置代理；代理地址不会被安装器打印或写入客户端配置。

5. 安装结束后检查本地 MCP `initialize` 与 `tools/list` 结果；按客户端提示刷新或重启 IDE。此检查不会调用 PingCode API，也不代表已完成真实 PingCode 联调。

## 安全边界

- 安装器只写 `PINGCODE_READ_ONLY=true`，不读取、收集或保存 Client ID、Client Secret、Token 或其他凭据。
- 不要求用户把凭据粘贴到对话、Skill 文件或共享工作区 MCP 配置。凭据只能由用户按本机安全方式通过环境变量提供。
- 不自动填充 `PINGCODE_ALLOWED_PROJECT_IDS`；空白名单不授予项目访问权限。服务器只读默认开启；不要把工具加入客户端自动批准列表。
- 不运行真实 PingCode API 调用。真实租户验证必须标记为 `INTEGRATION`，并且需要授权和隔离测试项目。
- 如果当前平台没有匹配 Release、摘要校验失败、压缩包路径不安全或 provenance 验证失败，停止安装并保留原程序和配置。

## 安装 Skill

- Kiro IDE：打开 Agent Steering & Skills 面板，选择 `Import a skill`，在本 Skill 发布到默认分支后导入公开 GitHub 子目录：<https://github.com/ihugang/pingcode-mcp/tree/main/skills/pingcode-mcp-setup>。
- 其他客户端：按 [references/ide-setup.md](references/ide-setup.md) 将整个 `pingcode-mcp-setup` 目录放入该客户端的 Skill 目录；必须保留 `SKILL.md`、`scripts/` 和 `references/`。

## 诊断

遇到配置路径、客户端 CLI 缺失或 MCP 工具未出现时，读取相应的客户端参考，只提供与当前客户端相关的步骤。不要通过覆盖配置文件或尝试多个客户端来排错。
