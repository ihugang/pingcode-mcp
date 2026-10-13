package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ihugang/pingcode-mcp-go/internal/pingcode"
	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

type listProjectsInput struct {
	ScopeType       *string `json:"scope_type,omitempty" jsonschema:"项目范围类型"`
	ScopeID         *string `json:"scope_id,omitempty" jsonschema:"项目范围 ID"`
	Keywords        *string `json:"keywords,omitempty" jsonschema:"项目名称关键词"`
	MemberType      *string `json:"member_type,omitempty" jsonschema:"成员类型；须与 member_id 同时提供"`
	MemberID        *string `json:"member_id,omitempty" jsonschema:"成员 ID；须与 member_type 同时提供"`
	Type            *string `json:"type,omitempty" jsonschema:"项目类型"`
	CreatedBetween  *string `json:"created_between,omitempty" jsonschema:"创建时间范围"`
	UpdatedBetween  *string `json:"updated_between,omitempty" jsonschema:"更新时间范围"`
	IncludeDeleted  bool    `json:"include_deleted,omitempty" jsonschema:"是否包含已删除项目"`
	IncludeArchived bool    `json:"include_archived,omitempty" jsonschema:"是否包含已归档项目"`
	PageSize        *int    `json:"page_size,omitempty" jsonschema:"每页条数，范围 0–100；0 使用默认值 30"`
	PageIndex       *int    `json:"page_index,omitempty" jsonschema:"从 0 开始的页码"`
}

type getProjectInput struct {
	ProjectID       string `json:"project_id" jsonschema:"项目 ID"`
	IncludeDeleted  bool   `json:"include_deleted,omitempty" jsonschema:"是否包含已删除项目"`
	IncludeArchived bool   `json:"include_archived,omitempty" jsonschema:"是否包含已归档项目"`
}

type listWorkItemsInput struct {
	ProjectID               string  `json:"project_id" jsonschema:"项目 ID"`
	Identifier              *string `json:"identifier,omitempty" jsonschema:"工作项编号"`
	TypeID                  *string `json:"type_id,omitempty" jsonschema:"工作项类型 ID 或系统类型枚举；枚举映射 INTEGRATION-PENDING，不做枚举校验"`
	ParentID                *string `json:"parent_id,omitempty" jsonschema:"父工作项 ID"`
	AssigneeID              *string `json:"assignee_id,omitempty" jsonschema:"负责人 ID"`
	StateID                 *string `json:"state_id,omitempty" jsonschema:"状态 ID"`
	PriorityID              *string `json:"priority_id,omitempty" jsonschema:"优先级 ID"`
	BugTypeID               *string `json:"bug_type_id,omitempty" jsonschema:"缺陷类型 ID"`
	TagID                   *string `json:"tag_id,omitempty" jsonschema:"标签 ID"`
	SprintID                *string `json:"sprint_id,omitempty" jsonschema:"迭代 ID"`
	BoardID                 *string `json:"board_id,omitempty" jsonschema:"看板 ID"`
	EntryID                 *string `json:"entry_id,omitempty" jsonschema:"看板入口 ID"`
	SwimlaneID              *string `json:"swimlane_id,omitempty" jsonschema:"泳道 ID"`
	PhaseID                 *string `json:"phase_id,omitempty" jsonschema:"阶段 ID"`
	ReleaseID               *string `json:"release_id,omitempty" jsonschema:"发布 ID"`
	CreatedBy               *string `json:"created_by,omitempty" jsonschema:"创建人 ID"`
	ParticipantID           *string `json:"participant_id,omitempty" jsonschema:"参与者 ID"`
	Keywords                *string `json:"keywords,omitempty" jsonschema:"工作项编号或标题关键词"`
	IncludePublicImageToken bool    `json:"include_public_image_token,omitempty" jsonschema:"是否包含公开图片令牌"`
	IncludeDeleted          bool    `json:"include_deleted,omitempty" jsonschema:"是否包含已删除工作项"`
	IncludeArchived         bool    `json:"include_archived,omitempty" jsonschema:"是否包含已归档工作项"`
	PageSize                *int    `json:"page_size,omitempty" jsonschema:"每页条数，范围 0–100；0 使用默认值 30"`
	PageIndex               *int    `json:"page_index,omitempty" jsonschema:"从 0 开始的页码"`
}

type getWorkItemInput struct {
	WorkItemID              string `json:"workitem_id" jsonschema:"工作项 ID 或编号"`
	ProjectID               string `json:"project_id" jsonschema:"本地授权上下文中的项目 ID"`
	IncludePublicImageToken bool   `json:"include_public_image_token,omitempty" jsonschema:"是否包含公开图片令牌"`
	IncludeDeleted          bool   `json:"include_deleted,omitempty" jsonschema:"是否包含已删除工作项"`
	IncludeArchived         bool   `json:"include_archived,omitempty" jsonschema:"是否包含已归档工作项"`
}

type createWorkItemInput struct {
	ProjectID         string         `json:"project_id" jsonschema:"目标项目 ID"`
	TypeID            string         `json:"type_id" jsonschema:"工作项类型 ID 或系统类型值"`
	Title             string         `json:"title" jsonschema:"工作项标题"`
	Description       *string        `json:"description,omitempty" jsonschema:"工作项描述"`
	StartAt           *int64         `json:"start_at,omitempty" jsonschema:"开始时间 Unix 时间戳"`
	EndAt             *int64         `json:"end_at,omitempty" jsonschema:"结束时间 Unix 时间戳"`
	PriorityID        *string        `json:"priority_id,omitempty" jsonschema:"优先级 ID"`
	StateID           *string        `json:"state_id,omitempty" jsonschema:"状态 ID"`
	AssigneeID        *string        `json:"assignee_id,omitempty" jsonschema:"负责人 ID"`
	ParentID          *string        `json:"parent_id,omitempty" jsonschema:"父工作项 ID"`
	SprintID          *string        `json:"sprint_id,omitempty" jsonschema:"迭代 ID"`
	ReleaseIDs        []string       `json:"release_ids,omitempty" jsonschema:"发布 ID 列表"`
	BoardID           *string        `json:"board_id,omitempty" jsonschema:"看板 ID"`
	EntryID           *string        `json:"entry_id,omitempty" jsonschema:"看板入口 ID"`
	SwimlaneID        *string        `json:"swimlane_id,omitempty" jsonschema:"泳道 ID"`
	StoryPoints       *float64       `json:"story_points,omitempty" jsonschema:"故事点数"`
	EstimatedWorkload *float64       `json:"estimated_workload,omitempty" jsonschema:"预估工作量"`
	RemainingWorkload *float64       `json:"remaining_workload,omitempty" jsonschema:"剩余工作量"`
	Properties        map[string]any `json:"properties,omitempty" jsonschema:"工作项自定义属性"`
	ParticipantIDs    []string       `json:"participant_ids,omitempty" jsonschema:"参与者 ID 列表"`
}

type updateWorkItemInput struct {
	WorkItemID        string         `json:"workitem_id" jsonschema:"工作项 ID 或编号"`
	ProjectID         string         `json:"project_id" jsonschema:"本地授权上下文中的项目 ID"`
	Title             *string        `json:"title,omitempty" jsonschema:"工作项标题"`
	Description       *string        `json:"description,omitempty" jsonschema:"工作项描述"`
	StartAt           *int64         `json:"start_at,omitempty" jsonschema:"开始时间 Unix 时间戳"`
	EndAt             *int64         `json:"end_at,omitempty" jsonschema:"结束时间 Unix 时间戳"`
	PriorityID        *string        `json:"priority_id,omitempty" jsonschema:"优先级 ID"`
	StateID           *string        `json:"state_id,omitempty" jsonschema:"状态 ID"`
	AssigneeID        *string        `json:"assignee_id,omitempty" jsonschema:"负责人 ID"`
	ParentID          *string        `json:"parent_id,omitempty" jsonschema:"父工作项 ID"`
	ReleaseIDs        []string       `json:"release_ids,omitempty" jsonschema:"发布 ID 列表"`
	BoardID           *string        `json:"board_id,omitempty" jsonschema:"看板 ID"`
	EntryID           *string        `json:"entry_id,omitempty" jsonschema:"看板入口 ID"`
	SwimlaneID        *string        `json:"swimlane_id,omitempty" jsonschema:"泳道 ID"`
	PhaseID           *string        `json:"phase_id,omitempty" jsonschema:"阶段 ID"`
	StoryPoints       *float64       `json:"story_points,omitempty" jsonschema:"故事点数"`
	EstimatedWorkload *float64       `json:"estimated_workload,omitempty" jsonschema:"预估工作量"`
	RemainingWorkload *float64       `json:"remaining_workload,omitempty" jsonschema:"剩余工作量"`
	Properties        map[string]any `json:"properties,omitempty" jsonschema:"工作项自定义属性"`
}

func registerTools(server *sdk.Server, api PingCodeAPI, policy *security.Policy) {
	sdk.AddTool[listProjectsInput, any](server, &sdk.Tool{
		Name: "pingcode_list_projects", Description: "列出当前项目白名单内可访问的 PingCode 项目。",
		InputSchema: schemaWithPagination[listProjectsInput](),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in listProjectsInput) (*sdk.CallToolResult, any, error) {
		pageSize, pageIndex, err := normalizedPagination(in.PageSize, in.PageIndex)
		if err != nil {
			return validationError(err.Error())
		}
		if !policy.AllowsAnyProject() {
			return nil, map[string]any{"page_size": pageSize, "page_index": pageIndex, "total": 0, "values": []json.RawMessage{}}, nil
		}
		page, err := api.ListProjects(ctx, pingcode.ListProjectsParams{
			ScopeType: derefString(in.ScopeType), ScopeID: derefString(in.ScopeID), Keywords: derefString(in.Keywords),
			MemberType: derefString(in.MemberType), MemberID: derefString(in.MemberID), Type: derefString(in.Type),
			CreatedBetween: derefString(in.CreatedBetween), UpdatedBetween: derefString(in.UpdatedBetween),
			IncludeDeleted: in.IncludeDeleted, IncludeArchived: in.IncludeArchived, PageSize: pageSize, PageIndex: pageIndex,
		})
		if err != nil {
			return toolError(err)
		}
		values := make([]json.RawMessage, 0, len(page.Values))
		for _, project := range page.Values {
			if !policy.AllowsProject(project.ID) {
				continue
			}
			raw, err := rawOrMarshal(project.Raw, project)
			if err != nil {
				return toolError(err)
			}
			values = append(values, raw)
		}
		return nil, map[string]any{"page_size": page.PageSize, "page_index": page.PageIndex, "total": len(values), "values": values}, nil
	})

	sdk.AddTool[getProjectInput, any](server, &sdk.Tool{
		Name: "pingcode_get_project", Description: "按项目 ID 读取白名单内的 PingCode 项目详情。",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in getProjectInput) (*sdk.CallToolResult, any, error) {
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return validationError("project_id must not be blank")
		}
		if err := policy.CheckProjectRead(projectID); err != nil {
			return toolError(err)
		}
		project, err := api.GetProject(ctx, projectID, pingcode.GetOptions{IncludeDeleted: in.IncludeDeleted, IncludeArchived: in.IncludeArchived})
		if err != nil {
			return toolError(err)
		}
		raw, err := rawOrMarshal(project.Raw, project)
		if err != nil {
			return toolError(err)
		}
		return nil, map[string]any{"project": raw}, nil
	})

	sdk.AddTool[listWorkItemsInput, any](server, &sdk.Tool{
		Name: "pingcode_list_work_items", Description: "按白名单项目及可选条件分页查询 PingCode 工作项。",
		InputSchema: schemaWithPagination[listWorkItemsInput](),
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in listWorkItemsInput) (*sdk.CallToolResult, any, error) {
		pageSize, pageIndex, err := normalizedPagination(in.PageSize, in.PageIndex)
		if err != nil {
			return validationError(err.Error())
		}
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return validationError("project_id must not be blank")
		}
		if err := policy.CheckProjectRead(projectID); err != nil {
			return toolError(err)
		}
		page, err := api.ListWorkItems(ctx, pingcode.ListWorkItemsParams{
			ProjectID: projectID, Identifier: derefString(in.Identifier), TypeID: derefString(in.TypeID),
			ParentID: derefString(in.ParentID), AssigneeID: derefString(in.AssigneeID), StateID: derefString(in.StateID),
			PriorityID: derefString(in.PriorityID), BugTypeID: derefString(in.BugTypeID), TagID: derefString(in.TagID),
			SprintID: derefString(in.SprintID), BoardID: derefString(in.BoardID), EntryID: derefString(in.EntryID),
			SwimlaneID: derefString(in.SwimlaneID), PhaseID: derefString(in.PhaseID), ReleaseID: derefString(in.ReleaseID),
			CreatedBy: derefString(in.CreatedBy), ParticipantID: derefString(in.ParticipantID), Keywords: derefString(in.Keywords),
			IncludePublicImageToken: in.IncludePublicImageToken, IncludeDeleted: in.IncludeDeleted,
			IncludeArchived: in.IncludeArchived, PageSize: pageSize, PageIndex: pageIndex,
		})
		if err != nil {
			return toolError(err)
		}
		values := make([]json.RawMessage, 0, len(page.Values))
		for _, item := range page.Values {
			raw, err := rawOrMarshal(item.Raw, item)
			if err != nil {
				return toolError(err)
			}
			values = append(values, raw)
		}
		return nil, map[string]any{"page_size": pageSize, "page_index": pageIndex, "total": page.Total, "values": values}, nil
	})

	sdk.AddTool[getWorkItemInput, any](server, &sdk.Tool{
		Name: "pingcode_get_work_item", Description: "按工作项 ID 读取指定白名单项目内的 PingCode 工作项。",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in getWorkItemInput) (*sdk.CallToolResult, any, error) {
		workitemID := strings.TrimSpace(in.WorkItemID)
		if workitemID == "" {
			return validationError("workitem_id must not be blank")
		}
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return validationError("project_id must not be blank")
		}
		if err := policy.CheckProjectRead(projectID); err != nil {
			return toolError(err)
		}
		item, err := api.GetWorkItem(ctx, workitemID, projectID, pingcode.GetOptions{
			IncludePublicImageToken: in.IncludePublicImageToken, IncludeDeleted: in.IncludeDeleted, IncludeArchived: in.IncludeArchived,
		})
		if err != nil {
			return toolError(err)
		}
		raw, err := rawOrMarshal(item.Raw, item)
		if err != nil {
			return toolError(err)
		}
		return nil, map[string]any{"work_item": raw}, nil
	})

	sdk.AddTool[createWorkItemInput, any](server, &sdk.Tool{
		Name: "pingcode_create_work_item", Description: "在白名单项目中创建单个 PingCode 工作项。",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in createWorkItemInput) (*sdk.CallToolResult, any, error) {
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return validationError("project_id must not be blank")
		}
		typeID := strings.TrimSpace(in.TypeID)
		if typeID == "" {
			return validationError("type_id must not be blank")
		}
		title := strings.TrimSpace(in.Title)
		if title == "" {
			return validationError("title must not be blank")
		}
		if err := policy.CheckProjectWrite(projectID); err != nil {
			return toolError(err)
		}
		item, err := api.CreateWorkItem(ctx, pingcode.CreateWorkItemRequest{
			ProjectID: projectID, TypeID: typeID, Title: title, Description: in.Description, StartAt: in.StartAt,
			EndAt: in.EndAt, PriorityID: in.PriorityID, StateID: in.StateID, AssigneeID: in.AssigneeID,
			ParentID: in.ParentID, SprintID: in.SprintID, ReleaseIDs: in.ReleaseIDs, BoardID: in.BoardID,
			EntryID: in.EntryID, SwimlaneID: in.SwimlaneID, StoryPoints: in.StoryPoints,
			EstimatedWorkload: in.EstimatedWorkload, RemainingWorkload: in.RemainingWorkload,
			Properties: in.Properties, ParticipantIDs: in.ParticipantIDs,
		})
		if err != nil {
			return toolError(err)
		}
		raw, err := rawOrMarshal(item.Raw, item)
		if err != nil {
			return toolError(err)
		}
		return nil, map[string]any{"work_item": raw}, nil
	})

	sdk.AddTool[updateWorkItemInput, any](server, &sdk.Tool{
		Name: "pingcode_update_work_item", Description: "更新白名单项目中的单个 PingCode 工作项。",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in updateWorkItemInput) (*sdk.CallToolResult, any, error) {
		workitemID := strings.TrimSpace(in.WorkItemID)
		if workitemID == "" {
			return validationError("workitem_id must not be blank")
		}
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return validationError("project_id must not be blank")
		}
		if !hasUpdateFields(in) {
			return validationError("update requires at least one field")
		}
		if err := policy.CheckProjectWrite(projectID); err != nil {
			return toolError(err)
		}
		item, err := api.UpdateWorkItem(ctx, workitemID, projectID, pingcode.UpdateWorkItemRequest{
			Title: in.Title, Description: in.Description, StartAt: in.StartAt, EndAt: in.EndAt,
			PriorityID: in.PriorityID, StateID: in.StateID, AssigneeID: in.AssigneeID, ParentID: in.ParentID,
			ReleaseIDs: in.ReleaseIDs, BoardID: in.BoardID, EntryID: in.EntryID, SwimlaneID: in.SwimlaneID,
			PhaseID: in.PhaseID, StoryPoints: in.StoryPoints, EstimatedWorkload: in.EstimatedWorkload,
			RemainingWorkload: in.RemainingWorkload, Properties: in.Properties,
		})
		if err != nil {
			return toolError(err)
		}
		raw, err := rawOrMarshal(item.Raw, item)
		if err != nil {
			return toolError(err)
		}
		return nil, map[string]any{"work_item": raw}, nil
	})
}

func schemaWithPagination[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	zero, hundred := float64(0), float64(100)
	schema.Properties["page_size"].Minimum = &zero
	schema.Properties["page_size"].Maximum = &hundred
	schema.Properties["page_index"].Minimum = &zero
	return schema
}

func normalizedPagination(pageSize, pageIndex *int) (int, int, error) {
	size, index := 30, 0
	if pageSize != nil {
		size = *pageSize
	}
	if pageIndex != nil {
		index = *pageIndex
	}
	if size < 0 || size > 100 {
		return 0, 0, inputError("page_size must be between 0 and 100")
	}
	if index < 0 {
		return 0, 0, inputError("page_index must be at least 0")
	}
	if size == 0 {
		size = 30
	}
	return size, index, nil
}

type inputError string

func (e inputError) Error() string { return string(e) }

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func hasUpdateFields(in updateWorkItemInput) bool {
	return in.Title != nil || in.Description != nil || in.StartAt != nil || in.EndAt != nil ||
		in.PriorityID != nil || in.StateID != nil || in.AssigneeID != nil || in.ParentID != nil ||
		len(in.ReleaseIDs) != 0 || in.BoardID != nil || in.EntryID != nil || in.SwimlaneID != nil ||
		in.PhaseID != nil || in.StoryPoints != nil || in.EstimatedWorkload != nil ||
		in.RemainingWorkload != nil || len(in.Properties) != 0
}

func rawOrMarshal(raw []byte, value any) (json.RawMessage, error) {
	if len(raw) > 0 && json.Valid(raw) {
		return append(json.RawMessage(nil), raw...), nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}
