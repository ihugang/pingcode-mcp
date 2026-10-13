# Contributing

感谢你对 PingCode MCP Server 的关注。本项目按 [MIT License](LICENSE) 发布，目前没有正式版本或 GitHub Release。

## 开发环境

- Go 1.25 或更新版本。
- 不需要真实 PingCode 凭据即可运行自动化测试。
- 不要读取、提交或分享 `.pingcode.env`、`.king-pingcode.env` 或其他本地凭据文件。

```sh
gofmt -l .
go vet ./...
go build ./...
go test ./...
go test -race ./...
```

格式检查应无输出。测试必须使用替身、`httptest` 或 MCP 协议 harness，不得依赖真实 PingCode 凭据；真实租户测试必须明确标记为 `INTEGRATION`，并记录授权范围和隔离测试项目。

## 变更范围和追踪

本项目使用 PRD、OpenSpec 和 Git 追踪需求。产品行为、权限边界、API 契约、工具 Schema、配置语义和外部依赖的变化必须先建立对应 OpenSpec change，并包含 proposal、design、tasks 和 spec delta，经负责人批准后再实现。

提交或 PR 描述应明确：

- 关联的 `docs/PRD.md` FR 编号；
- OpenSpec change 和 task；
- 未创建的 PingCode 需求 ID 使用 `PENDING`，不得编造 ID；
- 是否改变用户可见行为、权限边界、数据披露范围或真实写入行为。

仅文档或社区材料也应说明对应的 FR/OpenSpec 追踪。当前公开发布材料对应现有 `bootstrap-pingcode-mcp` change 的交付文档任务 5.1，并不新增产品能力。

## Pull Request 要求

1. 一个 PR 聚焦一个问题，不混入无关重构。
2. 描述行为变化、文档变化和风险；对于真实 PingCode 验证，注明 `INTEGRATION`，不要把 mock 结果写成联调结果。
3. 不提交 Token、Client Secret、Authorization Header、完整工作项正文或其他敏感值；示例只能使用变量名和占位值。
4. 说明测试命令及结果；至少覆盖适用的格式检查、`go vet ./...`、`go build ./...` 和 `go test ./...`。
5. 涉及写入策略时说明只读默认值和项目白名单影响，并确认没有增加删除或批量破坏性操作。
6. 未经维护者批准，不要创建正式发布版本或 GitHub Release；许可证已由仓库维护者确定为 MIT。

## 文档和配置示例

README、示例配置和 Issue/PR 模板不得包含真实凭据。程序不会自动解析 dotenv 文件；文档中若示范 `.pingcode.env`，必须说明它只用于本地环境变量导入，并保持在忽略规则中。

## 安全问题

不要通过公开 Issue 报告疑似漏洞或泄露的凭据。请阅读 [SECURITY.md](SECURITY.md)；安全漏洞应通过 GitHub Security Advisories 私下报告。发现本地凭据疑似泄露时，先停止使用并通知负责人轮换，不要把旧值或新值粘贴到任何公开渠道。
