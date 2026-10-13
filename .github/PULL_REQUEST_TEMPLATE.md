## 变更摘要

<!-- 说明解决的问题，以及是否改变用户可见行为。不要粘贴凭据或敏感工作项正文。 -->

## 需求与 Spec 追踪

- PRD FR：`<!-- 例如 FR-011；没有对应需求时说明文档范围 -->`
- OpenSpec change/task：`<!-- 例如 bootstrap-pingcode-mcp task 5.1；需求 ID 未创建时使用 PENDING -->`
- PingCode 需求 ID：`PENDING`

## 验证

- [ ] `gofmt -l .` 无输出
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go test ./...`
- [ ] 适用时运行 `go test -race ./...`
- [ ] 真实租户验证已标记为 `INTEGRATION`，并说明授权范围和隔离项目；否则确认未执行真实请求

验证命令和结果：

```text
<!-- 填写实际运行的命令和结果；不要包含 Secret、Token 或完整请求 URL。 -->
```

## 安全与发布检查

- [ ] 未提交 Token、Client Secret、Authorization Header 或本地凭据文件
- [ ] 示例配置只包含变量名和占位值
- [ ] 默认只读和项目白名单约束未被绕过
- [ ] 未增加删除或批量破坏性操作
- [ ] 涉及用户可见行为的变化已有获批 OpenSpec change
- [ ] 未选择许可证、创建远程仓库或推送发布（除非由维护者另行完成）
