# Tasks

PingCode 需求 ID：`PENDING`。

## 1. Spec

- [x] 1.1 记录负责人提出的安装下载应使用运行环境代理的需求，关联 PRD FR-003，并新增 spec delta 与实现设计。

## 2. Implementation

- [x] 2.1 显式通过 Python `ProxyHandler` 与环境代理设置构造下载 opener；不输出或保存代理值（FR-003；Spec：`installer-download-proxy`）。
- [x] 2.2 增加离线单元测试，验证代理环境进入下载 opener 且请求沿用 timeout（FR-003；Spec：`installer-download-proxy`）。
- [x] 2.3 在 Skill、IDE 参考和 README 说明代理变量与运行时配置位置（FR-003；Spec：`installer-download-proxy`）。

## 3. Validation and Release

- [x] 3.1 运行 Python 28 项离线单测、`gofmt -l .`、`go vet ./...`、`go build ./...`、`go test ./...`、Python 编译、`git diff --check` 和 GitHub token/Bearer 模式扫描；均通过。通过环境代理下载了 `SHA256SUMS`，7 条摘要与 GitHub Release asset digest 一致；darwin/arm64 压缩包响应头为 200，但读取 3 MiB 响应体持续无数据，故未完成解包安装（FR-003；PingCode ID：`PENDING`）。
- [x] 3.2 使用 Ruby 标准库 YAML parser 检查 `SKILL.md` frontmatter、必需字段和命名；通过。skill-creator `quick_validate.py` 因当前 Python 环境缺少 PyYAML 未能运行（FR-003；PingCode ID：`PENDING`）。
- [x] 3.3 发布 `v0.1.1` 补丁版本；CI run `38539822679` 与 Release run `38540306109` 均通过。Release 是正式版本，含 6 个平台二进制、`SHA256SUMS`、`VERSION.txt`；attestation 步骤通过，7 个资产摘要与 GitHub asset digest 一致（FR-003；PingCode ID：`PENDING`）。
