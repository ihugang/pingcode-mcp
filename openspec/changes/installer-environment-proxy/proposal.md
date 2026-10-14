# Proposal

## Why

安装器从 GitHub Releases 下载预编译程序和校验清单。处于代理网络的用户需要下载请求遵守启动安装器时配置的环境代理，否则 Release 安装可能无法完成。

## What Changes

- 明确使用 Python 标准库代理处理器读取当前进程代理设置，并保留 `NO_PROXY` 绕过行为。
- 在 Skill、IDE 参考和 README 中说明代理环境变量及其生效范围。
- 添加离线单元测试，验证 Release 下载 opener 使用环境代理配置。

## Traceability

- PRD：FR-003（跨 IDE Skill 与安装配置）。
- OpenSpec：`installer-environment-proxy` change；任务见 `tasks.md`。
- PingCode 需求 ID：`PENDING`。
