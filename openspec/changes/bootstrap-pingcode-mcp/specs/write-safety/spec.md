# Spec Delta

## Purpose

在本地配置层和每个工具调用边界限制 PingCode 数据访问，防止默认配置或调用方参数意外产生写入，并将项目白名单作为项目级读写授权范围。该能力不替代 PingCode 服务端自身的权限控制。

## ADDED Requirements

### Requirement: 默认只读并拒绝所有写操作

当 `PINGCODE_READ_ONLY` 未设置、为空或为 `true` 时，服务 SHALL 以只读模式运行，并在发出任何创建或更新请求前拒绝全部写操作；只有经校验的显式 `false` 才可解除只读门禁。

#### Scenario: 未设置只读配置
- **WHEN** 服务加载配置且未提供 `PINGCODE_READ_ONLY`
- **THEN** 只读模式为启用状态，任何创建或更新工具均在发出 PingCode 请求前返回 `READ_ONLY` 错误

#### Scenario: 只读模式中调用写工具
- **WHEN** `PINGCODE_READ_ONLY=true` 且调用创建或更新工作项工具
- **THEN** 请求被本地拒绝，PingCode 不收到写请求

#### Scenario: 显式关闭只读模式
- **WHEN** 配置为合法的 `PINGCODE_READ_ONLY=false` 且目标项目在白名单中
- **THEN** 写操作仍须通过项目白名单校验后才可发出

### Requirement: 项目白名单限制项目定向读取与写入

`PINGCODE_ALLOWED_PROJECT_IDS` SHALL 作为项目级访问白名单；详情读取、工作项列表、工作项详情及创建/更新 SHALL 仅允许指向精确匹配的白名单项目，空白名单 SHALL 不授权任何项目目标。

#### Scenario: 访问白名单项目
- **WHEN** 项目定向读取或写入的 `project_id` 与配置白名单中的 ID 精确匹配
- **THEN** 该调用可继续执行后续只读或写入策略检查

#### Scenario: 访问白名单以外项目
- **WHEN** 调用方请求的项目 ID 不在配置白名单中或白名单为空
- **THEN** 服务在发起该项目的 PingCode 请求前返回 `PROJECT_NOT_ALLOWED` 错误，且不返回该项目数据

#### Scenario: 列出项目
- **WHEN** 调用方使用 `pingcode_list_projects` 发现项目
- **THEN** 工具只向调用方返回配置白名单中的项目和分页数据；空白名单时不暴露任何项目资源

#### Scenario: 工作项 ID 与项目范围不匹配
- **WHEN** 工作项详情或更新请求中的 `workitem_id` 不属于调用方指定的白名单 `project_id`
- **THEN** 服务拒绝返回该工作项内容或执行更新，并返回安全的权限错误

### Requirement: 禁止删除与未授权批量写入

服务 SHALL NOT 提供删除操作；单次工作项创建或更新 SHALL 仅作用于调用方明确指定且通过白名单校验的单个项目和资源。

#### Scenario: 请求删除资源
- **WHEN** 客户端检查 MCP 工具集合或尝试调用删除能力
- **THEN** 服务不公布删除工具且不会发送删除请求

#### Scenario: 写入目标项目未明确或批量
- **WHEN** 创建或更新调用缺少唯一目标项目/资源，或包含多个项目的批量写入目标
- **THEN** 服务拒绝该调用，不发出 PingCode 写请求
