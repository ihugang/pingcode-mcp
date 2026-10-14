# Tasks

PingCode 需求 ID：`PENDING`。

## 1. Spec

- [x] 1.1 记录负责人提出的安装下载应使用运行环境代理的需求，关联 PRD FR-003，并新增 spec delta 与实现设计。

## 2. Implementation

- [x] 2.1 显式通过 Python `ProxyHandler` 与环境代理设置构造下载 opener；不输出或保存代理值（FR-003；Spec：`installer-download-proxy`）。
- [x] 2.2 增加离线单元测试，验证代理环境进入下载 opener 且请求沿用 timeout（FR-003；Spec：`installer-download-proxy`）。
- [x] 2.3 在 Skill、IDE 参考和 README 说明代理变量与运行时配置位置（FR-003；Spec：`installer-download-proxy`）。

## 3. Validation and Release

- [x] 3.1 运行 Python 28 项离线单测、`gofmt -l .`、`go vet ./...`、`go build ./...`、`go test ./...`、Python 编译、`git diff --check` 和 GitHub token/Bearer 模式扫描；目前通过。GitHub Release 下载只在临时目录尝试，端到端资产读取未完成（FR-003；PingCode ID：`PENDING`）。
- [x] 3.2 使用 Ruby 标准库 YAML parser 检查 `SKILL.md` frontmatter、必需字段和命名；通过。skill-creator `quick_validate.py` 因当前 Python 环境缺少 PyYAML 未能运行（FR-003；PingCode ID：`PENDING`）。
- [ ] 3.3 发布新的补丁版本，核对 GitHub CI/Release workflow 与 Release 资产；等待工作流完成。
