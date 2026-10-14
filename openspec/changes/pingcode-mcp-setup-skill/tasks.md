# Tasks

> 状态：负责人于 2026-10-14 明确批准本 change 并确认 FR-003 覆盖跨 IDE Skill/接入；任务按验收结果更新。PingCode 需求 ID：`PENDING`。

## 1. Gate

- [x] 1.1 负责人独立评审本 change 的 proposal、design 和 spec delta；确认 FR-003 覆盖跨 IDE Skill/接入指南后进入实现（关联 FR-003、FR-010；Spec：`cross-ide-setup-skill`；验收：负责人明确指示“执行吧”；PingCode ID：`PENDING`；已完成：PRD FR-003 同步更新）。

## 2. Skill Package

- [x] 2.1 新增符合 Agent Skills 开放规范的 `skills/pingcode-mcp-setup/SKILL.md`，包含有效 frontmatter、精确触发描述、安装器调用、Skill 与 MCP Server 的区分、环境识别和安全边界（关联 FR-003、FR-010；Spec：`cross-ide-setup-skill`；验证：skill-creator `quick_validate.py` 通过；PingCode ID：`PENDING`）。
- [x] 2.2 新增 `.github/workflows/release.yml`，在通过构建、vet、Go/Python 测试后由 `vX.Y.Z` tag 构建 darwin/linux/windows 的 amd64/arm64 资产，生成 SHA256SUMS、provenance attestation 并创建 GitHub Release（关联 FR-003、FR-011；Spec：`cross-ide-setup-skill`；验证：本地六目标交叉编译、归档命名及 6 个资产 SHA256SUMS 校验通过；GitHub tag/release/attestation 实际发布留待任务 2.6；PingCode ID：`PENDING`）。
- [x] 2.3 新增 `skills/pingcode-mcp-setup/scripts/install.py`，实现平台/版本匹配、release 资产下载、SHA-256 校验、安全解包、用户目录安装、客户端选择、dry-run、配置合并、同名冲突保护、原子写入和 MCP 本地初始化/工具发现验证；源码构建只能由用户显式选择；不得读取/写入凭据或调用 PingCode API（关联 FR-003、FR-010；Spec：`cross-ide-setup-skill`；验证：23 项离线单测覆盖 mock release 成功、损坏摘要不落盘、错误平台、取消、超时、配置合并和权限边界；本地真实二进制 `initialize/tools/list` 通过；PingCode ID：`PENDING`）。
- [x] 2.4 新增 `references/ide-setup.md`，涵盖 Kiro IDE、Codex、Claude Code、Cursor、VS Code/GitHub Copilot 的官方配置入口、用户级/workspace 级范围、stdio schema、release 安装和常见诊断；实施时记录逐项核对的官方资料链接（关联 FR-003；Spec：`cross-ide-setup-skill`；验证：逐项核对官方文档，检查示例 JSON/TOML；PingCode ID：`PENDING`）。
- [x] 2.5 检查所有示例保持只读、不给危险工具默认授权、不含凭据或用户机器路径；审阅校验失败、平台不支持、安装失败、用户取消、超时和权限拒绝路径（关联 FR-010；Spec：`cross-ide-setup-skill`；验证：安全审查、diff/秘密扫描；PingCode ID：`PENDING`）。
- [ ] 2.6 由负责人独立验收并发布稳定 `vX.Y.Z` tag；检查 GitHub Release 六种资产、SHA256SUMS 与 attestation，并从实际 GitHub Release 执行一次安装验证（关联 FR-003、FR-011；Spec：`cross-ide-setup-skill`；验证：真实 GitHub Release 和安装记录；该仓库规则要求 Agent 不得推送或发布；PingCode ID：`PENDING`）。

## 3. Project Documentation

- [x] 3.1 更新 README，说明 GitHub Skill 安装入口、支持客户端范围、Skill 安装与 MCP Server 配置的区别，并链接现有安装/凭据文档（关联 FR-003、FR-010；Spec：`cross-ide-setup-skill`；验证：文档路径和安装步骤逐项核对；PingCode ID：`PENDING`）。
- [x] 3.2 使用 Agent Skills validator 校验结构和 frontmatter；本地安装脚本与 MCP initialize/tools/list 验证通过（关联 FR-003；Spec：`cross-ide-setup-skill`；验证：skill-creator `quick_validate.py` 通过；PingCode ID：`PENDING`）。
- [x] 3.3 检查 `git diff`、`git status`、忽略文件及秘密扫描结果，确认没有本机凭据、用户配置或二进制进入发布内容；负责人独立验收后再决定提交与发布（关联 FR-010；Spec：`cross-ide-setup-skill`；验证：15 个变更/未跟踪文件高置信凭据模式扫描无命中，两个本地凭据文件模式均保持忽略；PingCode ID：`PENDING`）。
- [ ] 3.4 在 Kiro IDE 与至少一个其他客户端中验证 Skill/MCP 发现（关联 FR-003；Spec：`cross-ide-setup-skill`；验证：实际客户端截图或可复现记录；当前未操作用户 IDE，待负责人验收环境；PingCode ID：`PENDING`）。
