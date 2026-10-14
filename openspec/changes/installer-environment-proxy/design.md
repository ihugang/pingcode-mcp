# Design

## Context

安装器使用 Python 标准库下载 GitHub API 元数据、release 资产和 SHA-256 清单。负责人明确要求安装下载使用环境中定义的代理。现有实现通过 `urlopen` 隐式使用默认 opener；本 change 让代理选择显式可见、可测试。

## Decision

- 每次下载时通过 Python 标准库读取当前进程代理映射，显式构造 `urllib.request.ProxyHandler(proxies)` 并将其交给 `build_opener()`。
- HTTP(S) 请求优先读取对应协议的 `HTTP_PROXY`/`HTTPS_PROXY`（大小写均可）；协议变量缺失时回退到 `ALL_PROXY`。遵守 `NO_PROXY`。
- Python 标准库不提供 SOCKS 代理支持。若当前目标会使用 SOCKS 代理，且不在 `NO_PROXY` 中，安装器安全停止并提示配置 HTTP(S) 代理，避免误绕过代理直接联网。
- 不输出代理配置，不将其复制进 IDE 配置或 Skill 文件。网络错误继续使用不包含代理 URL 的通用提示。
- 不更改 release 来源、HTTPS 校验、SHA-256 校验、下载大小限制或配置写入流程。

## Validation

- 离线测试替身代理环境读取、opener 构造和下载调用，不发起网络请求。
- 运行 Python 单元测试、格式/静态检查以及仓库要求的 Go 校验。
- 通过 GitHub CI 和 Release workflow 验证发布资产。

## Traceability

- 关联 FR-003；PingCode 需求 ID：`PENDING`。
- 代码与测试对应任务 `2.1`、`2.2`；使用说明对应任务 `2.3`。
